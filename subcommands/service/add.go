package services

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Add(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("service add", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s <name> <key>=<value>...\n", flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() == 0 {
		return fmt.Errorf("no service specified")
	}

	var (
		service = flags.Arg(0)
		keys    = make(map[string]string, flags.NArg()-1)
	)

	for _, kv := range flags.Args()[1:] {
		key, val, found := strings.Cut(kv, "=")
		if !found || key == "" {
			return fmt.Errorf("invalid argument %q", kv)
		}
		keys[key] = val
	}

	sc, err := getClient(ctx)
	if err != nil {
		return err
	}

	if err := sc.SetServiceConfiguration(service, keys); err != nil {
		return err
	}
	if err := sc.SetServiceStatus(service, true); err != nil {
		return err
	}

	return nil
}
