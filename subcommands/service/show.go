package services

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"go.yaml.in/yaml/v3"
)

func Show(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		asJson      bool
		asYaml      bool
		showSecrets bool
	)

	flags := flag.NewFlagSet("service show", flag.ExitOnError)
	flags.BoolVar(&asJson, "json", false, "output in JSON format")
	flags.BoolVar(&asYaml, "yaml", false, "output in YAML format (default)")
	flags.BoolVar(&showSecrets, "secrets", false, "show secret values instead of ********")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	if flags.NArg() != 1 {
		return fmt.Errorf("invalid number of arguments, expected 1 but got %d", flags.NArg())
	}

	Service := flags.Arg(0)

	sc, err := getClient(ctx)
	if err != nil {
		return err
	}

	config, err := sc.GetServiceConfiguration(Service)
	if err != nil {
		return err
	}

	if asJson {
		err = json.NewEncoder(ctx.Stdout).Encode(map[string]any{Service: config})
	} else {
		err = yaml.NewEncoder(ctx.Stdout).Encode(map[string]any{Service: config})
	}
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
