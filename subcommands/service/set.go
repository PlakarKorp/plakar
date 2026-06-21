package services

import (
	"flag"
	"fmt"
	"maps"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

func Set(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	flags := flag.NewFlagSet("service set", flag.ExitOnError)
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

	if len(keys) == 0 {
		return nil
	}

	config, err := sc.GetServiceConfiguration(service)
	if err != nil {
		return err
	}

	maps.Copy(config, keys)
	if err := sc.SetServiceConfiguration(service, config); err != nil {
		return err
	}

	return nil
}
