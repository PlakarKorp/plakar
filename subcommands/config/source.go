package config

import (
	"flag"
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"

	"gopkg.in/yaml.v3"
)

type ConfigSourceCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigSourceCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("source", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	cmd.args = flags.Args()

	return nil
}

func (cmd *ConfigSourceCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	err := source_config(ctx, cmd.args)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func source_config(ctx *appcontext.AppContext, args []string) error {
	usage := "usage: plakar source [add|check|ls|ping|rm|set|unset]"
	cmd := "ls"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "add":
		usage := "usage: plakar source add <name> <location> [<key>=<value>, ...]"
		if len(args) < 2 {
			return fmt.Errorf(usage)
		}
		name, location := args[0], normalizeLocation(args[1])
		if ctx.Config.HasSource(name) {
			return fmt.Errorf("source %q already exists", name)
		}
		ctx.Config.Sources[name] = make(map[string]string)
		ctx.Config.Sources[name]["location"] = location
		for _, kv := range args[2:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Sources[name][key] = val
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "check":
		usage := "usage: plakar source check <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasSource(name) {
			return fmt.Errorf("source %q does not exist", name)
		}
		cfg, ok := ctx.Config.GetSource(name)
		if !ok {
			return fmt.Errorf("failed to retreive configuration for source %q", name)
		}
		imp, err := importer.NewImporter(ctx.GetInner(), ctx.ImporterOpts(), cfg)
		if err != nil {
			return err
		}
		err = imp.Close()
		if err != nil {
			ctx.GetLogger().Warn("error when closing source: %v", err)
		}
		return nil

	case "ls":
		usage := "usage: plakar source ls"
		if len(args) != 0 {
			return fmt.Errorf(usage)
		}
		return yaml.NewEncoder(ctx.Stdout).Encode(ctx.Config.Sources)

	case "ping":
		return fmt.Errorf("not implemented")

	case "rm":
		usage := "usage: plakar source rm <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasSource(name) {
			return fmt.Errorf("source %q does not exist", name)
		}
		delete(ctx.Config.Sources, name)
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "set":
		usage := "usage: plakar source set <name> [<key>=<value>, ...]"
		if len(args) == 0 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasSource(name) {
			return fmt.Errorf("source %q does not exist", name)
		}
		for _, kv := range args[1:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Sources[name][key] = val
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "unset":
		usage := "usage: plakar source unset <name> [<key>, ...]"
		if len(args) == 0 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasSource(name) {
			return fmt.Errorf("source %q does not exist", name)
		}
		for _, key := range args[1:] {
			if key == "location" {
				return fmt.Errorf("cannot unset location")
			}
			delete(ctx.Config.Sources[name], key)
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	default:
		return fmt.Errorf(usage)
	}
}
