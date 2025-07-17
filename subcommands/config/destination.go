package config

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"

	"gopkg.in/yaml.v3"
)

type ConfigDestinationCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigDestinationCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("destination", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	cmd.args = flags.Args()

	return nil
}

func (cmd *ConfigDestinationCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	err := destination_config(ctx, cmd.args)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func destination_config(ctx *appcontext.AppContext, args []string) error {
	usage := "usage: plakar destination [add|check|ls|ping|rm|set|unset]"
	cmd := "ls"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "add":
		usage := "usage: plakar destination add <name> <location> [<key>=<value>, ...]"
		if len(args) < 2 {
			return fmt.Errorf(usage)
		}
		name, location := args[0], normalizeLocation(args[1])
		if ctx.Config.HasDestination(name) {
			return fmt.Errorf("destination %q already exists", name)
		}
		ctx.Config.Destinations[name] = make(map[string]string)
		ctx.Config.Destinations[name]["location"] = location
		for _, kv := range args[2:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Destinations[name][key] = val
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "check":
		usage := "usage: plakar destination check <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasDestination(name) {
			return fmt.Errorf("destination %q does not exist", name)
		}
		cfg, ok := ctx.Config.GetDestination(name)
		if !ok {
			return fmt.Errorf("failed to retreive configuration for destination %q", name)
		}
		exp, err := exporter.NewExporter(ctx.GetInner(), cfg)
		if err != nil {
			return err
		}
		err = exp.Close()
		if err != nil {
			ctx.GetLogger().Warn("error when closing store: %v", err)
		}
		return nil

	case "ls":
		usage := "usage: plakar destination ls"
		if len(args) != 0 {
			return fmt.Errorf(usage)
		}
		return yaml.NewEncoder(ctx.Stdout).Encode(ctx.Config.Destinations)

	case "ping":
		return fmt.Errorf("not implemented")

	case "rm":
		usage := "usage: plakar destination rm <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasDestination(name) {
			return fmt.Errorf("destination %q does not exist", name)
		}
		delete(ctx.Config.Destinations, name)
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "set":
		usage := "usage: plakar destination set <name> [<key>=<value>, ...]"
		if len(args) == 0 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasDestination(name) {
			return fmt.Errorf("destination %q does not exist", name)
		}
		for _, kv := range args[1:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Destinations[name][key] = val
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "unset":
		usage := "usage: plakar destination unset <name> [<key>, ...]"
		if len(args) == 0 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasDestination(name) {
			return fmt.Errorf("destination %q does not exist", name)
		}
		for _, key := range args[1:] {
			if key == "location" {
				return fmt.Errorf("cannot unset location")
			}
			delete(ctx.Config.Destinations[name], key)
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	default:
		return fmt.Errorf(usage)
	}
}
