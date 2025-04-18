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
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"syscall"

	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/scheduler"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vmihailenco/msgpack/v5"
)

func init() {
	subcommands.Register(&Agent{}, "agent")
}

func daemonize(argv []string) error {
	binary, err := os.Executable()
	if err != nil {
		return err
	}

	procAttr := syscall.ProcAttr{}
	procAttr.Files = []uintptr{
		uintptr(syscall.Stdin),
		uintptr(syscall.Stdout),
		uintptr(syscall.Stderr),
	}
	procAttr.Env = append(os.Environ(),
		"REEXEC=1",
	)

	pid, err := syscall.ForkExec(binary, argv, &procAttr)
	if err != nil {
		return err
	}
	fmt.Printf("agent started with pid=%d\n", pid)
	os.Exit(0)
	return nil
}

func (cmd *Agent) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_foreground bool
	var opt_stop bool
	var opt_tasks string
	var opt_logfile string

	flags := flag.NewFlagSet("agent", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&opt_tasks, "tasks", "", "tasks configuration file")
	flags.StringVar(&cmd.prometheus, "prometheus", "", "prometheus exporter interface, e.g. 127.0.0.1:9090")
	flags.BoolVar(&opt_foreground, "foreground", false, "run in foreground")
	flags.StringVar(&opt_logfile, "log", "", "log file")
	flags.BoolVar(&opt_stop, "stop", false, "stop the agent")
	flags.Parse(args)

	if opt_stop {
		client, err := agent.NewClient(filepath.Join(ctx.CacheDir, "agent.sock"))
		if err != nil {
			return err
		}
		defer client.Close()

		retval, err := client.SendCommand(ctx, &AgentStop{}, map[string]string{})
		if err != nil {
			return err
		}
		os.Exit(retval)
	}

	var schedConfig *scheduler.Configuration
	if opt_tasks != "" {
		tmp, err := scheduler.ParseConfigFile(opt_tasks)
		if err != nil {
			return err
		}
		schedConfig = tmp
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
	}

	cmd.socketPath = filepath.Join(ctx.CacheDir, "agent.sock")
	cmd.schedConfig = schedConfig

	return nil
}

type AgentStop struct {
	subcommands.SubcommandBase
}

func (cmd *AgentStop) Name() string {
	return "agent-stop"
}

func (cmd *AgentStop) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	return 1, nil
}

func (cmd *AgentStop) Parse(ctx *appcontext.AppContext, args []string) error {
	return nil
}

type Agent struct {
	subcommands.SubcommandBase

	prometheus string
	socketPath string

	listener net.Listener

	schedConfig *scheduler.Configuration
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
	if cmd.schedConfig != nil {
		go func() {
			scheduler.NewScheduler(ctx, cmd.schedConfig).Run()
		}()
	}

	if err := cmd.ListenAndServe(ctx); err != nil {
		return 1, err
	}
	log.Println("Server gracefully stopped")
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

	if cmd.prometheus != "" {
		promlistener, err := net.Listen("tcp", cmd.prometheus)
		if err != nil {
			return fmt.Errorf("failed to bind prometheus listener: %w", err)
		}
		defer promlistener.Close()

		go func() {
			http.Handle("/metrics", promhttp.Handler())
			http.Serve(promlistener, nil)
		}()
	}

	var wg sync.WaitGroup

	for {
		conn, err := cmd.listener.Accept()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return nil
			}
			return fmt.Errorf("failed to accept connection: %w", err)
		}

		wg.Add(1)

		go func(_conn net.Conn) {
			defer _conn.Close()
			defer wg.Done()

			mu := sync.Mutex{}

			var encodingErrorOccurred bool
			encoder := msgpack.NewEncoder(_conn)
			decoder := msgpack.NewDecoder(_conn)

			// Create a context tied to the connection
			cancelCtx, cancel := context.WithCancel(context.Background())
			defer cancel()

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
			if !slices.Equal(clientvers, ourvers) {
				return
			}

			clientContext := appcontext.NewAppContextFrom(ctx)
			clientContext.SetContext(cancelCtx)
			defer clientContext.Close()

			write := func(packet agent.Packet) {
				if encodingErrorOccurred {
					return
				}
				select {
				case <-clientContext.GetContext().Done():
					return
				default:
					mu.Lock()
					if err := encoder.Encode(&packet); err != nil {
						encodingErrorOccurred = true
					}
					mu.Unlock()
				}
			}
			read := func(v interface{}) (interface{}, error) {
				if err := decoder.Decode(v); err != nil {
					if isDisconnectError(err) {
						cancel()
					}
					return nil, err
				}
				return v, nil
			}

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

			clientContext.Stdout = &CustomWriter{processFunc: processStdout}
			clientContext.Stderr = &CustomWriter{processFunc: processStderr}

			logger := logging.NewLogger(clientContext.Stdout, clientContext.Stderr)
			logger.EnableInfo()
			clientContext.SetLogger(logger)

			name, storeConfig, request, err := subcommands.DecodeRPC(decoder)
			if err != nil {
				if isDisconnectError(err) {
					fmt.Fprintf(os.Stderr, "Client disconnected during initial request\n")
					cancel() // Cancel the context on disconnect
					return
				}
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}

			// Attempt another decode to detect client disconnection during processing
			go func() {
				var tmp interface{}
				read(&tmp)
			}()

			subcommand := subcommands.Lookup(name)
			if subcommand == nil {
				fmt.Fprintf(os.Stderr, "unknown command received %s\n", name)
				return
			}

			if err := msgpack.Unmarshal(request, &cmd); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to decode client request: %s\n", err)
				return
			}

			var repo *repository.Repository
			fmt.Printf("In agent %x\n", subcommand.GetRepositorySecret())
			clientContext.SetSecret(subcommand.GetRepositorySecret())

			store, serializedConfig, err := storage.Open(storeConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open storage: %s\n", err)
				return
			}
			defer store.Close()

			repo, err = repository.New(clientContext, store, serializedConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open repository: %s\n", err)
				return
			}
			defer repo.Close()

			eventsDone := make(chan struct{})
			eventsChan := clientContext.Events().Listen()
			go func() {
				for evt := range eventsChan {
					serialized, err := events.Serialize(evt)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to serialize event: %s\n", err)
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

			status, err := subcommand.Execute(clientContext, repo)

			/*
				if status == 0 {
					SuccessInc(subcommand.Name())
				} else if status == 1 {
					FailureInc(subcommand.Name())
				} else {
					WarningInc(subcommand.Name())
				}
			*/

			clientContext.Close()
			<-eventsDone

			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			write(agent.Packet{
				Type:     "exit",
				ExitCode: status,
				Err:      errStr,
			})

		}(conn)
	}
}

type CustomWriter struct {
	processFunc func(string) // Function to handle the log lines
}

// Write implements the `io.Writer` interface.
func (cw *CustomWriter) Write(p []byte) (n int, err error) {
	cw.processFunc(string(p))
	return len(p), nil
}
