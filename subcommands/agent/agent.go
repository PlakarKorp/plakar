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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/scheduler"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/task"
	"github.com/PlakarKorp/plakar/utils"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &AgentTasksConfigure{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen, "agent", "tasks", "configure")
	subcommands.Register(func() subcommands.Subcommand { return &AgentTasksStart{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen, "agent", "tasks", "start")
	subcommands.Register(func() subcommands.Subcommand { return &AgentTasksStop{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen, "agent", "tasks", "stop")
	subcommands.Register(func() subcommands.Subcommand { return &AgentRestart{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen|subcommands.IgnoreVersion, "agent", "reload")
	subcommands.Register(func() subcommands.Subcommand { return &AgentRestart{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen|subcommands.IgnoreVersion, "agent", "restart")
	subcommands.Register(func() subcommands.Subcommand { return &AgentStop{} },
		subcommands.AgentSupport|subcommands.BeforeRepositoryOpen|subcommands.IgnoreVersion, "agent", "stop")
	subcommands.Register(func() subcommands.Subcommand { return &Agent{} },
		subcommands.BeforeRepositoryOpen, "agent", "start")
	subcommands.Register(func() subcommands.Subcommand { return &Agent{} },
		subcommands.BeforeRepositoryOpen, "agent")
}

var agentContextSingleton *AgentContext

func (cmd *Agent) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_foreground bool
	var opt_logfile string

	_, envAgentLess := os.LookupEnv("PLAKAR_AGENTLESS")
	if envAgentLess {
		return fmt.Errorf("agent can not be start when PLAKAR_AGENTLESS is set")
	}

	flags := flag.NewFlagSet("agent", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&cmd.prometheus, "prometheus", "", "prometheus exporter interface, e.g. 127.0.0.1:9090")
	flags.BoolVar(&opt_foreground, "foreground", false, "run in foreground")
	flags.StringVar(&opt_logfile, "log", "", "log file")
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

	ctx.GetLogger().Info("Plakar agent up")
	return nil
}

type schedulerState int8

var (
	AGENT_SCHEDULER_STOPPED schedulerState = 0
	AGENT_SCHEDULER_RUNNING schedulerState = 1
)

type AgentContext struct {
	agentCtx        *appcontext.AppContext
	schedulerCtx    *appcontext.AppContext
	schedulerConfig *scheduler.Configuration
	schedulerState  schedulerState
	mtx             sync.Mutex
}

type AgentStop struct {
	subcommands.SubcommandBase
}

func (cmd *AgentStop) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("agent stop", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)
	if flags.NArg() != 0 {
		return fmt.Errorf("too many arguments")
	}

	return nil
}

func (cmd *AgentStop) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if err := stop(); err != nil {
		return 1, err
	}
	return 0, nil
}

type AgentRestart struct {
	subcommands.SubcommandBase
}

func (cmd *AgentRestart) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("agent restart", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)
	if flags.NArg() != 0 {
		return fmt.Errorf("too many arguments")
	}

	return nil
}

func (cmd *AgentRestart) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if err := restart(); err != nil {
		return 1, fmt.Errorf("failed to restart agent: %w", err)
	}
	return 0, nil
}

func restart() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable path: %w", err)
	}
	return syscall.Exec(exePath, append([]string{exePath}, os.Args[1:]...), os.Environ())
}

type Agent struct {
	subcommands.SubcommandBase

	prometheus string
	socketPath string

	listener net.Listener
}

func (cmd *Agent) checkSocket() bool {
	conn, err := net.Dial("unix", cmd.socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (cmd *Agent) Close() error {
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

func (cmd *Agent) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	agentContextSingleton = &AgentContext{
		agentCtx: ctx,
	}

	if err := cmd.ListenAndServe(ctx); err != nil {
		return 1, err
	}
	ctx.GetLogger().Info("Server gracefully stopped")
	return 0, nil
}

func (cmd *Agent) ListenAndServe(ctx *appcontext.AppContext) error {
	if _, err := os.Stat(cmd.socketPath); err == nil {
		if !cmd.checkSocket() {
			cmd.Close()
		} else {
			return fmt.Errorf("already running")
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	var err error

	// Bind socket
	cmd.listener, err = net.Listen("unix", cmd.socketPath)
	if err != nil {
		return fmt.Errorf("failed to bind socket: %w", err)
	}
	defer os.Remove(cmd.socketPath)

	// Set socket permissions
	if err := os.Chmod(cmd.socketPath, 0600); err != nil {
		cmd.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	var promlistener net.Listener
	if cmd.prometheus != "" {
		promlistener, err = net.Listen("tcp", cmd.prometheus)
		if err != nil {
			return fmt.Errorf("failed to bind prometheus listener: %w", err)
		}
		defer promlistener.Close()

		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.Serve(promlistener, nil)
		}()
	}

	// close the listener when the context gets closed
	go func() {
		<-ctx.Done()
		if promlistener != nil {
			promlistener.Close()
		}
		cmd.listener.Close()
	}()

	var wg sync.WaitGroup

	for {
		conn, err := cmd.listener.Accept()
		if err != nil {
			wg.Wait()
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return nil
			}
			ctx.GetLogger().Warn("failed to accept connection: %v", err)
			return err
		}

		if err := ctx.ReloadConfig(); err != nil {
			ctx.GetLogger().Warn("could not load configuration: %v", err)
			return err
		}

		wg.Add(1)
		go handleClient(ctx, &wg, conn)
	}
}

func handleClient(ctx *appcontext.AppContext, wg *sync.WaitGroup, conn net.Conn) {
	defer conn.Close()
	defer wg.Done()

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
		clientContext.SetSecret(subcommand.GetRepositorySecret())
		store, serializedConfig, err = storage.Open(clientContext.GetInner(), storeConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to open storage: %v", err)
			fmt.Fprintf(clientContext.Stderr, "Failed to open storage: %s\n", err)
			return
		}
		defer store.Close()

		repo, err = repository.New(clientContext.GetInner(), clientContext.GetSecret(), store, serializedConfig)
		if err != nil {
			clientContext.GetLogger().Warn("Failed to open repository: %v", err)
			fmt.Fprintf(clientContext.Stderr, "Failed to open repository: %s\n", err)
			return
		}
		defer repo.Close()
	}

	eventsDone := make(chan struct{})
	eventsChan := clientContext.Events().Listen()
	go func() {
		for evt := range eventsChan {
			serialized, err := events.Serialize(evt)
			if err != nil {
				clientContext.GetLogger().Warn("Failed to serialize event: %v", err)
				fmt.Fprintf(clientContext.Stderr, "Failed to serialize event: %s\n", err)
				return
			}
			// Send the event to the client
			write(agent.Packet{
				Type: "event",
				Data: serialized,
			})
		}
		eventsDone <- struct{}{}
	}()

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
	<-eventsDone
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
