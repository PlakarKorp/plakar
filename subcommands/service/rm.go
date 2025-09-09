package services

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type ServiceRm struct {
	subcommands.SubcommandBase

	Service string
}

func (cmd *ServiceRm) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("service rm", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s <name>\n", flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() == 0 {
		return fmt.Errorf("no service specified")
	}

	if flags.NArg() > 1 {
		return fmt.Errorf("invalid argument %q", flags.Arg(1))
	}

	cmd.Service = flags.Arg(0)

	return nil
}

func (cmd *ServiceRm) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	sc, err := getClient(ctx)
	if err != nil {
		return 1, err
	}
	if err := sc.SetServiceStatus(cmd.Service, false); err != nil {
		return 1, err
	}
	if err := sc.SetServiceConfiguration(cmd.Service, make(map[string]string)); err != nil {
		return 1, err
	}

	return 0, nil
}
