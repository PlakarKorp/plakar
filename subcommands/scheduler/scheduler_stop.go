package scheduler

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/scheduler"
	"github.com/PlakarKorp/plakar/subcommands"
)

type SchedulerStop struct {
	subcommands.SubcommandBase
	socketPath string
}

func (cmd *SchedulerStop) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("scheduler stop", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)
	if flags.NArg() != 0 {
		return fmt.Errorf("too many arguments")
	}

	cmd.socketPath = filepath.Join(ctx.CacheDir, "scheduler.sock")
	return nil
}

func (cmd *SchedulerStop) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	cl, err := scheduler.NewClient(cmd.socketPath, false)
	if err != nil {
		if err == scheduler.ErrWrongVersion {
			return 1, fmt.Errorf("scheduler is running with a different version of plakar: %w", err)
		}
		return 1, fmt.Errorf("failed to connect to scheduler: %w", err)
	}
	defer cl.Close()

	return cl.Stop()
}
