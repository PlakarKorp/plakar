package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/internal/bootstrap"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/task"
	"github.com/PlakarKorp/plakar/utils"

	_ "github.com/PlakarKorp/plakar/subcommands/agent"
	_ "github.com/PlakarKorp/plakar/subcommands/archive"
	_ "github.com/PlakarKorp/plakar/subcommands/backup"
	_ "github.com/PlakarKorp/plakar/subcommands/cat"
	_ "github.com/PlakarKorp/plakar/subcommands/check"
	_ "github.com/PlakarKorp/plakar/subcommands/clone"
	_ "github.com/PlakarKorp/plakar/subcommands/config"
	_ "github.com/PlakarKorp/plakar/subcommands/create"
	_ "github.com/PlakarKorp/plakar/subcommands/diag"
	_ "github.com/PlakarKorp/plakar/subcommands/diff"
	_ "github.com/PlakarKorp/plakar/subcommands/digest"
	_ "github.com/PlakarKorp/plakar/subcommands/dup"
	_ "github.com/PlakarKorp/plakar/subcommands/help"
	_ "github.com/PlakarKorp/plakar/subcommands/info"
	_ "github.com/PlakarKorp/plakar/subcommands/locate"
	_ "github.com/PlakarKorp/plakar/subcommands/login"
	_ "github.com/PlakarKorp/plakar/subcommands/ls"
	_ "github.com/PlakarKorp/plakar/subcommands/maintenance"
	_ "github.com/PlakarKorp/plakar/subcommands/mount"
	_ "github.com/PlakarKorp/plakar/subcommands/pkg"
	_ "github.com/PlakarKorp/plakar/subcommands/prune"
	_ "github.com/PlakarKorp/plakar/subcommands/ptar"
	_ "github.com/PlakarKorp/plakar/subcommands/restore"
	_ "github.com/PlakarKorp/plakar/subcommands/rm"
	_ "github.com/PlakarKorp/plakar/subcommands/scheduler"
	_ "github.com/PlakarKorp/plakar/subcommands/server"
	_ "github.com/PlakarKorp/plakar/subcommands/service"
	_ "github.com/PlakarKorp/plakar/subcommands/ui"
	_ "github.com/PlakarKorp/plakar/subcommands/version"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	_ "github.com/PlakarKorp/integration-fs/importer"
	_ "github.com/PlakarKorp/integration-fs/storage"
	_ "github.com/PlakarKorp/integration-ptar/storage"
	_ "github.com/PlakarKorp/integration-stdio/exporter"
	_ "github.com/PlakarKorp/integration-stdio/importer"
	_ "github.com/PlakarKorp/integration-tar/importer"
)

func entryPoint() int {
	cfg := bootstrap.NewConfigContext(os.Args)
	defer func() {
		if err := cfg.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cleanup error: %v\n", cfg.ProgramName, err)
		}
	}()

	pipeline := bootstrap.NewPipeline(
		bootstrap.NewConfigStage(),
		bootstrap.NewProfilingStage(),
		bootstrap.NewSecurityStage(),
		bootstrap.NewRepositoryStage(),
		bootstrap.NewSignalStage(),
	)

	if err := pipeline.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", cfg.ProgramName, err)
		return 1
	}

	if cfg.ShouldExit {
		return cfg.ExitCode
	}

	cmd := cfg.Command
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "%s: no command resolved\n", cfg.ProgramName)
		return 1
	}

	if err := cmd.Parse(cfg.App, cfg.CommandArgs); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", cfg.ProgramName, err)
		return 1
	}

	cmd.SetCWD(cfg.App.CWD)
	cmd.SetCommandLine(cfg.App.CommandLine)

	runWithoutAgent := cfg.Options.Agentless || cmd.GetFlags()&subcommands.AgentSupport == 0

	repo := cfg.Repository.Repository()
	var status int
	var err error

	if runWithoutAgent {
		status, err = task.RunCommand(cfg.App, cmd, repo, "@agentless")
	} else {
		status, err = agent.ExecuteRPC(cfg.App, cfg.CommandName, cmd, cfg.StoreConfig)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", cfg.ProgramName, utils.SanitizeText(err.Error()))
		if errors.Is(err, agent.ErrWrongVersion) {
			fmt.Fprintln(os.Stderr, "To stop the current agent, run:")
			fmt.Fprintln(os.Stderr, "\t$ plakar agent stop")
		}
	}

	if cfg.Profiling != nil {
		if perr := cfg.Profiling.Finalize(cfg); perr != nil {
			return 1
		}
	}

	return status
}

func main() {
	os.Exit(entryPoint())
}
