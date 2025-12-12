/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
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
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/ls"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"

	"github.com/vmihailenco/msgpack/v5"
)

type Cached struct {
	subcommands.SubcommandBase

	socketPath string
	listener   net.Listener

	teardown time.Duration

	jobMtx   sync.Mutex
	jobQueue map[uuid.UUID](chan jobReq)
}

type jobReq struct {
	ch (chan bool)
}

func (cmd *Cached) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_foreground bool
	var opt_logfile string

	_, envAgentLess := os.LookupEnv("PLAKAR_AGENTLESS")
	if envAgentLess {
		return fmt.Errorf("agent can not be started when PLAKAR_AGENTLESS is set")
	}

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

	clientContext := appcontext.NewAppContextFrom(ctx)
	defer clientContext.Close()

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
		case <-clientContext.Done():
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

	stdinchan := make(chan agent.Packet, 1)
	defer close(stdinchan)

	processStderr := func(data string) {
		write(agent.Packet{
			Type: "stderr",
			Data: []byte(data),
		})
	}

	name, storeConfig, request, err := subcommands.DecodeRPC(decoder)
	if err != nil {
		if isDisconnectError(err) {
			ctx.GetLogger().Warn("client disconnected during initial request")
			return
		}
		ctx.GetLogger().Warn("Failed to decode RPC: %v", err)
		fmt.Fprintf(clientContext.Stderr, "%s\n", err)
		return
	}

	// Attempt another decode to detect client disconnection during processing
	go func() {
		for {
			var pkt agent.Packet
			if err := decoder.Decode(&pkt); err != nil {
				if !isDisconnectError(err) {
					processStderr(fmt.Sprintf("failed to decode: %s", err))
				}
				clientContext.Close()
				return
			}
			if pkt.Type == "stdin" {
				stdinchan <- pkt
			}
		}
	}()

	subcommand := &ls.Ls{}
	if err := msgpack.Unmarshal(request, subcommand); err != nil {
		ctx.GetLogger().Warn("Failed to decode client request: %v", err)
		fmt.Fprintf(clientContext.Stderr, "Failed to decode client request: %s\n", err)
		return
	}

	ctx.GetLogger().Info("%s at %s", strings.Join(name, " "), storeConfig["location"])

	// XXX: It's kinda unfortunate that we have to open the store every time we
	// do a request, I'd love to be able to memoize it for the duration of the
	// rebuildJob goroutine but sadly we need the repo ID to index into it ...
	var serializedConfig []byte
	store, serializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		ctx.GetLogger().Warn("Failed to open storage: %v", err)
		return
	}
	defer store.Close(ctx)

	if err := setupSecret(ctx, subcommand, storeConfig, serializedConfig); err != nil {
		ctx.GetLogger().Warn("Failed to setup secret: %v", err)
		return
	}

	repo, err := repository.NewNoRebuild(ctx.GetInner(), ctx.GetSecret(), store, serializedConfig)
	if err != nil {
		ctx.GetLogger().Warn("Failed to open repository: %v", err)
		return
	}
	repo.Close()

	repoID := repo.Configuration().RepositoryID

	cmd.jobMtx.Lock()
	// Is there already a job goroutine running for this repo:
	if _, ok := cmd.jobQueue[repoID]; ok {
		cmd.jobMtx.Unlock()

		j := jobReq{
			ch: make(chan bool, 1),
		}

		cmd.jobQueue[repoID] <- j
		<-j.ch

		// All good our job ended.
	} else {
		cmd.jobQueue[repo.Configuration().RepositoryID] = make(chan jobReq, 1)
		cmd.jobMtx.Unlock()

		cmd.jobQueue[repoID] <- jobReq{
			ch: make(chan bool, 1),
		}

		go cmd.rebuildJob(clientContext, subcommand, repo)

		j := jobReq{
			ch: make(chan bool, 1),
		}

		cmd.jobQueue[repoID] <- j
		<-j.ch
	}

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	write(agent.Packet{
		Type:     "exit",
		ExitCode: -1, // XXX
		Err:      errStr,
	})
}

func (cmd *Cached) rebuildJob(ctx *appcontext.AppContext, subcommand *ls.Ls, repo *repository.Repository) {
	repoID := repo.Configuration().RepositoryID

	// Safe to access without a lock because if we are inside this function it
	// was definitely inserted before.
	jobChan := cmd.jobQueue[repoID]

jobLoop:
	for {
		select {
		case job := <-jobChan:
			// XXX: This is wrong as this reinstantiates a cache every time, it
			// kneeds an API change on kloset side. It'll do for now though.
			repo.RebuildState()

			// Notify that we ended
			close(job.ch)

		// Debounce a bit to avoid halting and creating too many jobs.
		case <-time.After(5 * time.Second):
			break jobLoop
		}
	}

	// Ok, no more job enqueued let's just remove ourself.
	cmd.jobMtx.Lock()
	delete(cmd.jobQueue, repoID)
	cmd.jobMtx.Unlock()
}
