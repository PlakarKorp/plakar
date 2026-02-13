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

package scheduler

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/scheduler"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/vmihailenco/msgpack/v5"
)

var schedulerContextSingleton *SchedulerContext

func (cmd *SchedulerStart) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_foreground bool
	var opt_logfile string
	var opt_tasks string

	flags := flag.NewFlagSet("scheduler start", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&opt_foreground, "foreground", false, "run in foreground")
	flags.StringVar(&opt_logfile, "log", "", "log file")
	flags.StringVar(&opt_tasks, "tasks", "", "tasks configuration file")
	flags.Parse(args)
	if flags.NArg() != 0 {
		return fmt.Errorf("too many arguments")
	}

	if opt_tasks == "" {
		return fmt.Errorf("no tasks configuration file provided")
	}

	var rd io.Reader
	if strings.HasPrefix(opt_tasks, "http://") || strings.HasPrefix(opt_tasks, "https://") {
		resp, err := http.Get(opt_tasks)
		if err != nil {
			return fmt.Errorf("failed to download configuration file from %q: %w", opt_tasks, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("failed to download configuration file from %q: status code %d", opt_tasks, resp.StatusCode)
		}

		rd = resp.Body
	} else {
		fp, err := os.Open(opt_tasks)
		if err != nil {
			return fmt.Errorf("failed to open %q: %w", opt_tasks, err)
		}
		defer fp.Close()
		rd = fp
	}

	configBytes, err := io.ReadAll(rd)
	if err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}
	_, err = scheduler.ParseConfigBytes(configBytes)
	if err != nil {
		return err
	}
	cmd.schedConfigBytes = configBytes

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

	ctx.GetLogger().Info("Plakar scheduler up")
	cmd.socketPath = filepath.Join(ctx.CacheDir, "scheduler.sock")

	return nil
}

type schedulerState int8

var (
	AGENT_SCHEDULER_STOPPED schedulerState = 0
	AGENT_SCHEDULER_RUNNING schedulerState = 1
)

type SchedulerContext struct {
	agentCtx        *appcontext.AppContext
	schedulerCtx    *appcontext.AppContext
	schedulerConfig *scheduler.Configuration
	schedulerState  schedulerState
	mtx             sync.Mutex
}

type SchedulerStart struct {
	subcommands.SubcommandBase
	socketPath       string
	schedConfigBytes []byte
}

func (cmd *SchedulerStart) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	schedulerContextSingleton = &SchedulerContext{
		agentCtx: ctx,
	}

	configureTasks(cmd.schedConfigBytes)
	startTasks()

	if err := cmd.ListenAndServe(ctx); err != nil {
		return 1, err
	}

	ctx.GetLogger().Info("Scheduler gracefully stopped")
	return 0, nil
}

func (cmd *SchedulerStart) ListenAndServe(ctx *appcontext.AppContext) error {
	listener, err := net.Listen("unix", cmd.socketPath)
	if err != nil {
		return fmt.Errorf("failed to bind the socket: %w", err)
	}

	var inflight atomic.Int64
	var nextID atomic.Int64

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

			// this can never happen, right?
			//if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
			//	return nil
			//}

			// TODO: we should retry / wait and retry on
			// some errors, not everything is fatal.

			return err
		}

		inflight.Add(1)

		go func() {
			// it's better to have this already in place,
			// even though we're not using IDs right now.
			_ = nextID.Add(1)
			defer func() {
				inflight.Add(-1)
			}()

			if err := ctx.ReloadConfig(); err != nil {
				ctx.GetLogger().Warn("could not load configuration: %v", err)
			}

			handleClient(ctx, conn)
		}()
	}
}

func handleClient(_ *appcontext.AppContext, conn net.Conn) {
	defer conn.Close()

	encoder := msgpack.NewEncoder(conn)
	decoder := msgpack.NewDecoder(conn)

	var clientvers []byte
	if err := decoder.Decode(&clientvers); err != nil {
		return
	}

	ourvers := []byte(utils.GetVersion())
	if err := encoder.Encode(ourvers); err != nil {
		return
	}

	// depending on packet, call proper handler

	var request scheduler.Request
	if err := decoder.Decode(&request); err != nil {
		return
	}

	var response scheduler.Response
	switch request.Type {
	case "stop":
		if _, err := terminate(); err != nil {
			response.ExitCode = 1
			response.Err = err.Error()
		} else {
			response.ExitCode = 0
		}
	default:
		response.ExitCode = 1
		response.Err = fmt.Sprintf("unknown command: %s", request.Type)
	}

	if err := encoder.Encode(response); err != nil {
		return
	}
}

func startTasks() (int, error) {
	schedulerContextSingleton.mtx.Lock()
	defer schedulerContextSingleton.mtx.Unlock()

	if schedulerContextSingleton.schedulerConfig == nil {
		return 1, fmt.Errorf("agent scheduler does not have a configuration")
	}

	if schedulerContextSingleton.schedulerState&AGENT_SCHEDULER_RUNNING != 0 {
		return 1, fmt.Errorf("agent scheduler already running")
	}

	// this needs to execute in the agent context, not the client context
	schedulerContextSingleton.schedulerCtx = appcontext.NewAppContextFrom(schedulerContextSingleton.agentCtx)
	go scheduler.NewScheduler(schedulerContextSingleton.schedulerCtx, schedulerContextSingleton.schedulerConfig).Run()

	schedulerContextSingleton.schedulerState = AGENT_SCHEDULER_RUNNING

	return 0, nil
}

func terminate() (int, error) {
	return 0, stop()
}

func configureTasks(schedConfigBytes []byte) (int, error) {
	schedConfig, err := scheduler.ParseConfigBytes(schedConfigBytes)
	if err != nil {
		return 1, err
	}

	schedulerContextSingleton.mtx.Lock()
	defer schedulerContextSingleton.mtx.Unlock()

	if schedulerContextSingleton.schedulerCtx != nil {
		schedulerContextSingleton.schedulerCtx.Cancel(nil)
		schedulerContextSingleton.schedulerCtx = appcontext.NewAppContextFrom(schedulerContextSingleton.agentCtx)
		go scheduler.NewScheduler(schedulerContextSingleton.schedulerCtx, schedConfig).Run()
	}

	schedulerContextSingleton.schedulerConfig = schedConfig
	return 0, nil
}
