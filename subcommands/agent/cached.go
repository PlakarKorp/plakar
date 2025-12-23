/*
 * Copyright (c) 2025 Mathieu Masson <mathieu@plakar.io>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package agent

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/cached"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"

	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	if runtime.GOOS != "windows" {
		subcommands.Register(func() subcommands.Subcommand { return &Cached{} },
			subcommands.BeforeRepositoryOpen, "cached")
	}
}

type Cached struct {
	subcommands.SubcommandBase

	socketPath string
	listener   net.Listener

	teardown time.Duration

	jobMtx   sync.Mutex
	jobQueue map[uuid.UUID](chan jobReq)
}

type jobReq struct {
	ch (chan error)
}

func (cmd *Cached) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_foreground bool
	var opt_logfile string

	flags := flag.NewFlagSet("cached", flag.ExitOnError)
	flags.StringVar(&opt_logfile, "log", "", "log file")
	flags.BoolVar(&opt_foreground, "foreground", false, "run in foreground")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.DurationVar(&cmd.teardown, "teardown", 5*time.Second, "delay before tearing down cached")
	flags.Parse(args)
	if flags.NArg() != 0 {
		return fmt.Errorf("too many arguments")
	}

	if !opt_foreground && os.Getenv("REEXEC") == "" {
		err := daemonize(os.Args)
		return err
	}

	if opt_logfile != "" {
		f, err := os.OpenFile(opt_logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		ctx.GetLogger().SetOutput(f)
	} else if !opt_foreground {
		if err := setupSyslog(ctx); err != nil {
			return err
		}
	}

	cmd.socketPath = filepath.Join(ctx.CacheDir, "cached.sock")

	cmd.jobMtx = sync.Mutex{}
	cmd.jobQueue = make(map[uuid.UUID]chan jobReq)

	return nil
}

func (cmd *Cached) Close() error {
	if cmd.listener != nil {
		cmd.listener.Close()
	}
	if err := os.Remove(cmd.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (cmd *Cached) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	// Since we are detaching, we loose all stack traces, with no possibility
	// to recover them, try to log them to a known location.
	crashLog := filepath.Join(ctx.GetInner().CacheDir, "crash-cached.log")
	f, err := os.OpenFile(crashLog, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return 1, err
	}

	debug.SetCrashOutput(f, debug.CrashOptions{})

	// Safe to ignore here.
	f.Close()

	if err := cmd.ListenAndServe(ctx); err != nil {
		return 1, err
	}

	ctx.GetLogger().Info("Server gracefully stopped")
	return 0, nil
}

func (cmd *Cached) ListenAndServe(ctx *appcontext.AppContext) error {
	lock, err := agent.LockedFile(cmd.socketPath + ".cached-lock")
	if err != nil {
		return fmt.Errorf("failed to obtain lock")
	}
	conn, err := net.Dial("unix", cmd.socketPath)
	if err == nil {
		lock.Unlock()
		conn.Close()
		return fmt.Errorf("cached already running")
	}
	os.Remove(cmd.socketPath)

	listener, err := net.Listen("unix", cmd.socketPath)
	lock.Unlock()

	if err != nil {
		return fmt.Errorf("failed to bind the socket: %w", err)
	}

	cancelled := false
	go func() {
		<-ctx.Done()
		cancelled = true
		listener.Close()
	}()

	var inflight atomic.Int64
	var nextID atomic.Int64
	for {
		conn, err := listener.Accept()
		if err != nil {
			if cancelled {
				return ctx.Err()
			}

			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return nil
			}
			// TODO: we should retry / wait and retry on
			// some errors, not everything is fatal.
			return err
		}

		inflight.Add(1)

		go func() {
			myid := nextID.Add(1)
			defer func() {
				n := inflight.Add(-1)
				if n == 0 {
					time.Sleep(cmd.teardown)
					if nextID.Load() == myid && inflight.Load() == 0 {
						listener.Close()
					}
				}
			}()

			if err := ctx.ReloadConfig(); err != nil {
				ctx.GetLogger().Warn("could not load configuration: %v", err)
			}

			cmd.handleCachedClient(ctx, conn)
		}()
	}
}

func (cmd *Cached) handleCachedClient(ctx *appcontext.AppContext, conn net.Conn) {
	defer conn.Close()

	mu := sync.Mutex{}

	var encodingErrorOccurred bool
	encoder := msgpack.NewEncoder(conn)
	decoder := msgpack.NewDecoder(conn)

	// handshake
	var (
		clientvers []byte
		ourvers    = []byte(utils.GetVersion())
	)
	if err := decoder.Decode(&clientvers); err != nil {
		return
	}
	if err := encoder.Encode(ourvers); err != nil {
		return
	}

	write := func(packet agent.Packet) {
		if encodingErrorOccurred {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
			mu.Lock()
			if err := encoder.Encode(&packet); err != nil {
				encodingErrorOccurred = true
				ctx.GetLogger().Warn("client write error: %v", err)
			}
			mu.Unlock()
		}
	}

	_, storeConfig, request, err := subcommands.DecodeRPC(decoder)
	if err != nil {
		if isDisconnectError(err) {
			ctx.GetLogger().Warn("client disconnected during initial request")
			return
		}
		ctx.GetLogger().Warn("Failed to decode RPC: %v", err)
		return
	}

	// Attempt another decode to detect client disconnection during processing
	go func() {
		for {
			var pkt agent.Packet
			if err := decoder.Decode(&pkt); err != nil {
				return
			}
		}
	}()

	subcommand := &cached.CachedReq{}
	if err := msgpack.Unmarshal(request, subcommand); err != nil {
		ctx.GetLogger().Warn("Failed to decode client request: %v", err)
		return
	}

	ctx.GetLogger().Info("cached rebuild request for %s", subcommand.RepoID)

	// Is there already a job goroutine running for this repo:
	var jq chan jobReq
	var ok bool
	cmd.jobMtx.Lock()
	if jq, ok = cmd.jobQueue[subcommand.RepoID]; !ok {
		cmd.jobQueue[subcommand.RepoID] = make(chan jobReq, 1)
		jq = cmd.jobQueue[subcommand.RepoID]

		err = cmd.rebuildJob(ctx, jq, subcommand, storeConfig)
	}

	cmd.jobMtx.Unlock()

	if err == nil {
		j := jobReq{
			ch: make(chan error, 1),
		}

		jq <- j
		err = <-j.ch
	}

	errStr := ""
	exitCode := 0
	if err != nil {
		errStr = err.Error()
		exitCode = -1
	}

	write(agent.Packet{
		Type:     "exit",
		ExitCode: exitCode,
		Err:      errStr,
	})
}

func (cmd *Cached) rebuildJob(ctx *appcontext.AppContext, jobChan chan jobReq, subcommand *cached.CachedReq, storeConfig map[string]string) error {
	var serializedConfig []byte
	store, serializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}

	if err := setupSecret(ctx, subcommand, storeConfig, serializedConfig); err != nil {
		return fmt.Errorf("failed to setup secret: %w", err)
	}

	repo, err := repository.NewNoRebuild(ctx.GetInner(), ctx.GetSecret(), store, serializedConfig)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	if subcommand.RepoID != repo.Configuration().RepositoryID {
		return fmt.Errorf("invalid uuid given %q repository id is %q", subcommand.RepoID.String(), repo.Configuration().RepositoryID.String())
	}

	go func() {
		defer store.Close(ctx)
		defer repo.Close()

		repoID := repo.Configuration().RepositoryID

	jobLoop:
		for {
			select {
			case job := <-jobChan:
				// XXX: This is wrong as this reinstantiates a cache every time, it
				// kneeds an API change on kloset side. It'll do for now though.
				err := repo.RebuildState()

				// Notify that we ended
				job.ch <- err
				close(job.ch)

			// Debounce a bit to avoid halting and creating too many jobs.
			case <-ctx.Done():
				break jobLoop
			}
		}

		// Ok, no more job enqueued let's just remove ourself.
		cmd.jobMtx.Lock()
		delete(cmd.jobQueue, repoID)
		cmd.jobMtx.Unlock()
	}()

	return nil

}

func setupSecret(ctx *appcontext.AppContext, cmd subcommands.Subcommand, storeConfig map[string]string, storageConfig []byte) error {
	config, err := storage.NewConfigurationFromWrappedBytes(storageConfig)
	if err != nil {
		return err
	}

	if config.Encryption == nil {
		return nil
	}

	getKey := func() ([]byte, error) {
		if key := cmd.GetRepositorySecret(); key != nil {
			return key, nil
		}

		passphrase, ok := storeConfig["passphrase"]
		if !ok {
			cmd, ok := storeConfig["passphrase_cmd"]
			if !ok {
				return nil, fmt.Errorf("no passphrase specified")
			}
			passphrase, err = utils.GetPassphraseFromCommand(cmd)
			if err != nil {
				return nil, fmt.Errorf("failed to read passphrase from command: %w", err)
			}
		}

		key, err := encryption.DeriveKey(config.Encryption.KDFParams, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("failed to derive key: %w", err)
		}
		return key, nil
	}

	key, err := getKey()
	if err != nil {
		return err
	}
	if !encryption.VerifyCanary(config.Encryption, key) {
		return fmt.Errorf("failed to verify key")
	}

	ctx.SetSecret(key)
	return nil
}

func isDisconnectError(err error) bool {
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
