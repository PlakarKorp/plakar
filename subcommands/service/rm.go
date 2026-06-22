package services

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Rm(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
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

	service := flags.Arg(0)

	sc, err := getClient(ctx)
	if err != nil {
		return err
	}
	if err := sc.SetServiceStatus(service, false); err != nil {
		return err
	}
	if err := sc.SetServiceConfiguration(service, make(map[string]string)); err != nil {
		return err
	}

	return nil
}
