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
		flags.PrintDefaults()
	}

	flags.Parse(args)
	cmd.args = flags.Args()

	return nil
}

func (cmd *ConfigSourceCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	err := configure(ctx, "source", cmd.args)
	if err != nil {
		return 1, err
	}
	return 0, nil
}
