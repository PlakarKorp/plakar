package services

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Unset(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("service unset", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s <name> <key>...\n", flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() == 0 {
		return fmt.Errorf("no service specified")
	}

	var (
		service = flags.Arg(0)
		keys    = flags.Args()[1:]
	)

	sc, err := getClient(ctx)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	config, err := sc.GetServiceConfiguration(service)
	if err != nil {
		return err
	}

	for _, key := range keys {
		delete(config, key)
	}

	if err := sc.SetServiceConfiguration(service, config); err != nil {
		return err
	}

	return nil
}
