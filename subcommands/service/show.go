package services

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"go.yaml.in/yaml/v3"
)

type ServiceShow struct {
	subcommands.SubcommandBase

	AsJson      bool
	AsYaml      bool
	ShowSecrets bool
	Service     string
}

func (cmd *ServiceShow) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("service show", flag.ExitOnError)
	flags.BoolVar(&cmd.AsJson, "json", false, "output in JSON format")
	flags.BoolVar(&cmd.AsYaml, "yaml", false, "output in YAML format (default)")
	flags.BoolVar(&cmd.ShowSecrets, "secrets", false, "show secret values instead of ********")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	if flags.NArg() != 1 {
		return fmt.Errorf("invalid number of arguments, expected 1 but got %d", flags.NArg())
	}

	cmd.Service = flags.Arg(0)
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
