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
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	psync "github.com/PlakarKorp/plakar/subcommands/sync"
	"github.com/PlakarKorp/plakar/task"
	"github.com/PlakarKorp/plakar/utils"

	"github.com/vmihailenco/msgpack/v5"
)

type AgentStart struct {
	subcommands.SubcommandBase

	socketPath string
	listener   net.Listener

	teardown time.Duration
}

func (cmd *AgentStart) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_foreground bool
	var opt_logfile string

	_, envAgentLess := os.LookupEnv("PLAKAR_AGENTLESS")
	if envAgentLess {
		return fmt.Errorf("agent can not be started when PLAKAR_AGENTLESS is set")
	}

	flags := flag.NewFlagSet("agent start", flag.ExitOnError)
	flags.StringVar(&opt_logfile, "log", "", "log file")
	flags.BoolVar(&opt_foreground, "foreground", false, "run in foreground")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.DurationVar(&cmd.teardown, "teardown", 5*time.Second, "delay before tearing down the agent")
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

	cmd.socketPath = filepath.Join(ctx.CacheDir, "agent.sock")
	return nil
}

func (cmd *AgentStart) Close() error {
	if cmd.listener != nil {
		cmd.listener.Close()
	}
	if err := os.Remove(cmd.socketPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func isDisconnectError(err error) bool {
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func (cmd *AgentStart) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	// Since we are detaching, we loose all stack traces, with no possibility
	// to recover them, try to log them to a known location.
	crashLog := filepath.Join(ctx.GetInner().CacheDir, "crash.log")
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

func (cmd *AgentStart) ListenAndServe(ctx *appcontext.AppContext) error {
	lock, err := agent.LockedFile(cmd.socketPath + ".agent-lock")
	if err != nil {
		return fmt.Errorf("failed to obtain lock")
	}
	conn, err := net.Dial("unix", cmd.socketPath)
	if err == nil {
		lock.Unlock()
		conn.Close()
		return fmt.Errorf("agent already running")
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

			handleClient(ctx, conn)
		}()
	}
}

func handleClient(ctx *appcontext.AppContext, conn net.Conn) {
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

	processStdout := func(data string) {
		write(agent.Packet{
			Type: "stdout",
			Data: []byte(data),
		})
	}

	processStderr := func(data string) {
		write(agent.Packet{
			Type: "stderr",
			Data: []byte(data),
		})
	}

	clientContext.Stdin = &CustomReader{stdinchan, encoder, &mu, ctx, nil}
	clientContext.Stdout = &CustomWriter{processFunc: processStdout}
	clientContext.Stderr = &CustomWriter{processFunc: processStderr}

	logger := logging.NewLogger(clientContext.Stdout, clientContext.Stderr)
	logger.EnableInfo()
	clientContext.SetLogger(logger)

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

	subcommand, _, _ := subcommands.Lookup(name)
	if subcommand == nil {
		ctx.GetLogger().Warn("unknown command received: %s", name)
		fmt.Fprintf(clientContext.Stderr, "unknown command received %s\n", name)
		return
	}
	if err := msgpack.Unmarshal(request, &subcommand); err != nil {
		ctx.GetLogger().Warn("Failed to decode client request: %v", err)
		fmt.Fprintf(clientContext.Stderr, "Failed to decode client request: %s\n", err)
		return
	}

	if subcommand.GetLogInfo() {
		clientContext.GetLogger().EnableInfo()
	}
	clientContext.GetLogger().EnableTracing(subcommand.GetLogTraces())
	clientContext.CWD = subcommand.GetCWD()
	clientContext.CommandLine = subcommand.GetCommandLine()

	ctx.GetLogger().Info("%s at %s", strings.Join(name, " "), storeConfig["location"])

	var store storage.Store
	var repo *repository.Repository

	if subcommand.GetFlags()&subcommands.BeforeRepositoryOpen != 0 {
		// nop
	} else if subcommand.GetFlags()&subcommands.BeforeRepositoryWithStorage != 0 {
		repo, err = repository.Inexistent(clientContext.GetInner(), storeConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to open raw storage: %v", err)
			fmt.Fprintf(clientContext.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
			return
		}
		defer repo.Close()
	} else {
		var serializedConfig []byte
		store, serializedConfig, err = storage.Open(clientContext.GetInner(), storeConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to open storage: %v", err)
			fmt.Fprintf(clientContext.Stderr, "Failed to open storage: %s\n", err)
			return
		}
		defer store.Close(ctx)
		err := setupSecret(clientContext, subcommand, storeConfig, serializedConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to setup secret: %v", err)
			fmt.Fprintf(clientContext.Stderr, "Failed to stup secret: %s\n", err)
			return
		}

		repo, err = repository.New(clientContext.GetInner(), clientContext.GetSecret(), store, serializedConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to open repository: %v", err)
			fmt.Fprintf(clientContext.Stderr, "Failed to open repository: %s\n", err)
			return
		}
		defer repo.Close()
	}

	if synccmd, ok := subcommand.(*psync.Sync); ok {
		if err := setupPeerSecret(clientContext, synccmd); err != nil {
			clientContext.GetLogger().Warn("Failed to setup peer secret: %v", err)
			fmt.Fprintf(clientContext.Stderr, "Failed to setup peer secret: %s\n", err)
			return
		}
	}

	status, err := task.RunCommand(clientContext, subcommand, repo, "@agent")

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	write(agent.Packet{
		Type:     "exit",
		ExitCode: status,
		Err:      errStr,
	})

	clientContext.Close()
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

func setupPeerSecret(ctx *appcontext.AppContext, cmd *psync.Sync) error {
	storeConfig, err := ctx.Config.GetRepository(cmd.PeerRepositoryLocation)
	if err != nil {
		return fmt.Errorf("peer repository: %w", err)
	}

	peerStore, peerStoreSerializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return fmt.Errorf("failed to open peer storage: %w", err)
	}
	peerStore.Close(ctx)

	peerStoreConfig, err := storage.NewConfigurationFromWrappedBytes(peerStoreSerializedConfig)
	if err != nil {
		return fmt.Errorf("failed to parse peer configuration: %w", err)
	}

	if peerStoreConfig.Encryption == nil {
		return nil
	}

	getKey := func() ([]byte, error) {
		if key := cmd.PeerRepositorySecret; key != nil {
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

		key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("failed to derive key: %w", err)
		}
		return key, nil
	}

	key, err := getKey()
	if err != nil {
		return err
	}
	if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
		return fmt.Errorf("failed to verify key")
	}

	cmd.PeerRepositorySecret = key
	return nil
}

type CustomWriter struct {
	processFunc func(string) // Function to handle the log lines
}

// Write implements the `io.Writer` interface.
func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	cw.processFunc(string(p))
	return len(p), nil
}

type CustomReader struct {
	ch      <-chan agent.Packet
	encoder *msgpack.Encoder
	mu      *sync.Mutex
	ctx     context.Context
	buf     []byte
}

func (cr *CustomReader) Read(p []byte) (n int, err error) {
	for {
		if len(cr.buf) != 0 {
			n := copy(p, cr.buf)
			cr.buf = cr.buf[n:]
			return n, nil
		}

		req := agent.Packet{Type: "stdin"}
		cr.mu.Lock()
		if err := cr.encoder.Encode(&req); err != nil {
			cr.mu.Unlock()
			return 0, err
		}
		cr.mu.Unlock()

		select {
		case pkt, ok := <-cr.ch:
			if !ok {
				return 0, io.EOF
			}
			if pkt.Eof {
				return 0, io.EOF
			}
			if pkt.Err != "" {
				return 0, fmt.Errorf("%s", pkt.Err)
			}
			cr.buf = append(cr.buf, pkt.Data...)
		case <-cr.ctx.Done():
			return 0, io.EOF
		}
	}
}
