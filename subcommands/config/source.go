package config

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type ConfigSourceCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigSourceCmd) Parse(ctx *appcontext.AppContext, args []string) error {
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
	cmd.args = flags.Args()
	return nil
}

func (cmd *ConfigSourceCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	err := dispatchSubcommand(ctx, "source", cmd.args[0], cmd.args[1:])
	if err != nil {
		return 1, err
	}
	return 0, nil
}
