package config

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type ConfigKlosetCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigKlosetCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("kloset", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	cmd.args = flags.Args()

	return nil
}

func (cmd *ConfigKlosetCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	err := cmd_kloset_config(ctx, cmd.args)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func cmd_kloset_config(ctx *appcontext.AppContext, args []string) error {
	usage := "usage: plakar kloset [add|check|default|ls|ping|set|unset]"
	cmd := "ls"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "add":
		usage := "usage: plakar kloset add <name> <location> [<key>=<value>, ...]"
		if len(args) < 2 {
			return fmt.Errorf(usage)
		}
		name, location := args[0], args[1]
		if ctx.Config.HasRepository(name) {
			return fmt.Errorf("kloset %q already exists", name)
		}
		ctx.Config.Repositories[name] = make(map[string]string)
		ctx.Config.Repositories[name]["location"] = location
		for _, kv := range args[2:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Repositories[name][key] = val
		}
		return ctx.Config.Save()

	case "check":
		usage := "usage: plakar kloset check <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("kloset %q does not exists", name)
		}
		_, err := storage.New(ctx.GetInner(), ctx.Config.Repositories[name])
		if err != nil {
			return err
		}
		return nil

	case "default":
		usage := "usage: plakar kloset default <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("kloset %q does not exists", name)
		}
		ctx.Config.DefaultRepository = name
		return ctx.Config.Save()

	case "ls":
		usage := "usage: plakar remote ls"
		if len(args) != 0 {
			return fmt.Errorf(usage)
		}
		var list []string
		for name, _ := range ctx.Config.Repositories {
			list = append(list, name)
		}
		sort.Strings(list)
		for i, name := range list {
			entry := ctx.Config.Repositories[name]
			if i != 0 {
				fmt.Fprint(ctx.Stdout, "\n")
			}
			if ctx.Config.DefaultRepository == name {
				fmt.Fprintf(ctx.Stdout, "; default\n")
			}
			fmt.Fprintf(ctx.Stdout, "[%s]\nlocation=%s\n", name, entry["location"])
			var keys []string
			for key, _ := range entry {
				if key != "location" {
					keys = append(keys, key)
				}
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(ctx.Stdout, "%s=%s\n", key, entry[key])
			}
		}
		return nil

	case "ping":
		usage := "usage: plakar kloset ping <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("kloset %q does not exists", name)
		}
		_, _, err := storage.Open(ctx.GetInner(), ctx.Config.Repositories[name])
		if err != nil {
			return err
		}
		return nil

	case "rm":
		usage := "usage: plakar kloset rm <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("kloset %q does not exist", name)
		}
		delete(ctx.Config.Repositories, name)
		return ctx.Config.Save()

	case "set":
		usage := "usage: plakar kloset set <name> [<key>=<value>, ...]"
		if len(args) == 0 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("kloset %q does not exists", name)
		}
		for _, kv := range args[1:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Repositories[name][key] = val
		}
		return ctx.Config.Save()

	case "unset":
		usage := "usage: plakar kloset unset <name> [<key>, ...]"
		if len(args) == 0 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRepository(name) {
			return fmt.Errorf("kloset %q does not exists", name)
		}
		for _, key := range args[1:] {
			if key == "location" {
				return fmt.Errorf("cannot unset location")
			}
			delete(ctx.Config.Repositories[name], key)
		}
		return ctx.Config.Save()

	default:
		return fmt.Errorf(usage)
	}
}
