package config

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func ConfigSource(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("source", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s add <name> <location> [<option>=<value>]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s check <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s import [-config <location>] [-overwrite] [-rclone] [<section>...]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s ping <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s rm <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s set <name> [<option>=<value>...]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s show [<name>...]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s unset <name> <option>...\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	if flags.NArg() == 0 {
		return fmt.Errorf("no action specified")
	}

	return dispatchSubcommand(ctx, "source", flags.Args()[0], flags.Args()[1:])
}
