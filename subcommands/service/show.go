package services

import (
	"encoding/json"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

type ServiceShow struct {
	subcommands.SubcommandBase

	AsJson      bool
	AsYaml      bool
	ShowSecrets bool
	Service     string
}

func (cmd *ServiceShow) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "service show [OPTIONS] <name>",
	}
	c.Flags().BoolVar(&cmd.AsJson, "json", false, "output in JSON format")
	c.Flags().BoolVar(&cmd.AsYaml, "yaml", false, "output in YAML format (default)")
	c.Flags().BoolVar(&cmd.ShowSecrets, "secrets", false, "show secret values instead of ********")
	return c
}

func (cmd *ServiceShow) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) != 1 {
		return fmt.Errorf("invalid number of arguments, expected 1 but got %d", len(rest))
	}

	cmd.Service = rest[0]
	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *ServiceShow) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	sc, err := getClient(ctx)
	if err != nil {
		return 1, err
	}

	config, err := sc.GetServiceConfiguration(cmd.Service)
	if err != nil {
		return 1, err
	}

	if cmd.AsJson {
		err = json.NewEncoder(ctx.Stdout).Encode(map[string]any{cmd.Service: config})
	} else {
		err = yaml.NewEncoder(ctx.Stdout).Encode(map[string]any{cmd.Service: config})
	}
	if err != nil {
		return 1, fmt.Errorf("failed to encode config: %w", err)
	}

	return 0, nil

}
