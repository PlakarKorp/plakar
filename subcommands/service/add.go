package services

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type ServiceAdd struct {
	subcommands.SubcommandBase

	Service string
	Keys    map[string]string
}

func (cmd *ServiceAdd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("service add", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s <name> <key>=<value>...\n", flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() == 0 {
		return fmt.Errorf("no service specified")
	}

	cmd.Service = flags.Arg(0)
	cmd.Keys = make(map[string]string)

	for _, kv := range flags.Args()[1:] {
		key, val, found := strings.Cut(kv, "=")
		if !found || key == "" {
			return fmt.Errorf("invalid argument %q", kv)
		}
		cmd.Keys[key] = val
	}

	return nil
}

func (cmd *ServiceAdd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	sc, err := getClient(ctx)
	if err != nil {
		return 1, err
	}

	if err := sc.SetServiceConfiguration(cmd.Service, cmd.Keys); err != nil {
		return 1, err
	}
	if err := sc.SetServiceStatus(cmd.Service, true); err != nil {
		return 1, err
	}

	return 0, nil
}
