package services

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type ServiceAdd struct {
	subcommands.SubcommandBase

	Service string
	Keys    map[string]string
}

func (cmd *ServiceAdd) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "service add",
	}
}

func (cmd *ServiceAdd) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) == 0 {
		return fmt.Errorf("no service specified")
	}

	cmd.Service = rest[0]
	cmd.Keys = make(map[string]string)

	for _, kv := range rest[1:] {
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
