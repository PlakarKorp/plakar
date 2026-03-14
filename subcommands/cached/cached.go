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

package cached

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
	"time"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cached"
	"github.com/PlakarKorp/plakar/subcommands"
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

	runningJobs chan int
}

type jobReq struct {
	stateID objects.MAC
	ch      chan error
}

const (
	newJob  = 1
	jobDone = -1
)

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
			ctx.GetLogger().Error("failed to setup syslog: %s", err)
			ctx.GetLogger().Error("will discard all future logs")
			ctx.GetLogger().SetOutput(io.Discard)
		}
	}

	cmd.socketPath = filepath.Join(ctx.CacheDir, "cached.sock")

	cmd.jobMtx = sync.Mutex{}
	cmd.jobQueue = make(map[uuid.UUID]chan jobReq)
	cmd.runningJobs = make(chan int)

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
	f, err := os.OpenFile(crashLog, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
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

// Background task dealing with the teardown, basically anything running sends a
// message over the channel, when taks is done we get another message. If the
// teardown time passes and nothing is in flight we are free to stop, otherwise
// someone is running and we need to stay alive. This is not a perfect "last
// request + teardown seconds" implementation but it's close enough for our need
// and is simpler. This is conceptually a waitgroup, except we can't use a
// waitgroup as it has one special property (you can't reincrement the semaphore
// while a Wait() is in progress) that our use case would transgress.
func (cmd *Cached) Watcher(listener net.Listener) {
	var inflight int

	for {
		select {
		case n := <-cmd.runningJobs:
			inflight += n
		case <-time.After(cmd.teardown):
			if inflight == 0 {
				listener.Close()
			}
		}
	}
}

func (cmd *Cached) ListenAndServe(ctx *appcontext.AppContext) error {
	lock, err := cached.LockedFile(cmd.socketPath + ".cached-lock")
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

	go cmd.Watcher(listener)

	cancelled := false
	go func() {
		<-ctx.Done()
		cancelled = true
		listener.Close()
	}()

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

		cmd.runningJobs <- newJob
		go func() {
			defer func() { cmd.runningJobs <- jobDone }()

			if err := ctx.ReloadConfig(); err != nil {
				ctx.GetLogger().Warn("could not load configuration: %v", err)
			}

			cmd.handleCachedClient(ctx, conn)
		}()
	}

}

func (cmd *Cached) handleCachedClient(ctx *appcontext.AppContext, conn net.Conn) {
	defer conn.Close()

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

	pkt := &cached.RequestPkt{}
	if err := decoder.Decode(pkt); err != nil {
		if isDisconnectError(err) {
			ctx.GetLogger().Warn("client disconnected during initial request")
			return
		}
		ctx.GetLogger().Warn("Failed to decode RPC: %v", err)
		return
	}

	ctx.GetLogger().Info("cached rebuild request for %s", pkt.RepoID)

	// Is there already a job goroutine running for this repo:
	var jq chan jobReq
	var ok bool
	var err error
	cmd.jobMtx.Lock()
	if jq, ok = cmd.jobQueue[pkt.RepoID]; !ok {
		jq = make(chan jobReq, 1024)
		err = cmd.rebuildJob(ctx, jq, pkt.RepoID, pkt.Secret, pkt.StoreConfig)

		if err == nil {
			cmd.jobQueue[pkt.RepoID] = jq
		}
	}
	cmd.jobMtx.Unlock()

	if err == nil {
		j := jobReq{
			stateID: pkt.StateID,
		}

		if !pkt.FireAndForget {
			j.ch = make(chan error, 1)
		}

		jq <- j

		if !pkt.FireAndForget {
			err = <-j.ch
		}
	}

	errStr := ""
	exitCode := 0
	if err != nil {
		errStr = err.Error()
		exitCode = -1
	}

	response := &cached.ResponsePkt{
		ExitCode: exitCode,
		Err:      errStr,
	}
	if err := encoder.Encode(&response); err != nil {
		ctx.GetLogger().Warn("client write error: %v", err)
	}
}

func (cmd *Cached) rebuildJob(ctx *appcontext.AppContext, jobChan chan jobReq, repoID uuid.UUID, secret []byte, storeConfig map[string]string) error {
	var serializedConfig []byte
	store, serializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return fmt.Errorf("failed to open storage: %w", err)
	}

	key, err := getSecret(ctx, secret, serializedConfig)
	if err != nil {
		return fmt.Errorf("failed to setup secret: %w", err)
	}

	repo, err := repository.NewNoRebuild(ctx.GetInner(), key, store, serializedConfig, false)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	if repoID != repo.Configuration().RepositoryID {
		return fmt.Errorf("invalid uuid given %q repository id is %q", repoID.String(), repo.Configuration().RepositoryID.String())
	}

	go func() {
		defer store.Close(ctx)
		defer repo.Close()

		repoID := repo.Configuration().RepositoryID

	jobLoop:
		for {
			select {
			case job := <-jobChan:
				cmd.runningJobs <- newJob
				var err error
				if job.stateID == objects.NilMac {
					err = repo.RebuildState()
				} else {
					err = repo.IngestStateFile(job.stateID)
				}

				// Notify that we ended
				if job.ch != nil {
					job.ch <- err
					close(job.ch)
				}

				cmd.runningJobs <- jobDone

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

func getSecret(ctx *appcontext.AppContext, secret []byte, storageConfig []byte) ([]byte, error) {
	config, err := storage.NewConfigurationFromWrappedBytes(storageConfig)
	if err != nil {
		return nil, err
	}

	if config.Encryption == nil {
		return nil, nil
	}

	key := secret
	if !encryption.VerifyCanary(config.Encryption, key) {
		return nil, fmt.Errorf("failed to verify key")
	}

	return key, nil
}

func isDisconnectError(err error) bool {
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
