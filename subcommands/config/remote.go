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

type ConfigRemoteCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigRemoteCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("remote", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	cmd.args = flags.Args()

	return nil
}

func (cmd *ConfigRemoteCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {

	err := cmd_remote_config(ctx, cmd.args)
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func cmd_remote_config(ctx *appcontext.AppContext, args []string) error {
	usage := "usage: plakar remote [add|check|ls|ping|rm|set|unset]"
	cmd := "ls"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "add":
		usage := "usage: plakar remote add <name> <location> [<key>=<value>, ...]"
		if len(args) < 2 {
			return fmt.Errorf(usage)
		}
		name, location := args[0], args[1]
		if ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q already exists", name)
		}
		ctx.Config.Remotes[name] = make(map[string]string)
		ctx.Config.Remotes[name]["location"] = location
		for _, kv := range args[2:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Remotes[name][key] = val
		}
		return ctx.Config.Save()

	case "check":
		usage := "usage: plakar remote check <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("kloset %q does not exists", name)
		}
		_, err := storage.New(ctx.GetInner(), ctx.Config.Remotes[name])
		if err != nil {
			return err
		}
		return nil

	case "ls":
		usage := "usage: plakar remote ls"
		if len(args) != 0 {
			return fmt.Errorf(usage)
		}
		var list []string
		for name, _ := range ctx.Config.Remotes {
			list = append(list, name)
		}
		sort.Strings(list)
		for i, name := range list {
			entry := ctx.Config.Remotes[name]
			if i != 0 {
				fmt.Fprint(ctx.Stdout, "\n")
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
		usage := "usage: plakar remote ping <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}
		_, _, err := storage.Open(ctx.GetInner(), ctx.Config.Remotes[name])
		if err != nil {
			return err
		}
		return nil

	case "rm":
		usage := "usage: plakar remote rm <name>"
		if len(args) != 1 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exist", name)
		}
		delete(ctx.Config.Remotes, name)
		return ctx.Config.Save()

	case "set":
		usage := "usage: plakar remote set <name> [<key>=<value>, ...]"
		if len(args) == 0 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}
		for _, kv := range args[1:] {
			key, val, found := strings.Cut(kv, "=")
			if !found {
				return fmt.Errorf(usage)
			}
			if key == "" {
				return fmt.Errorf(usage)
			}
			ctx.Config.Remotes[name][key] = val
		}
		return ctx.Config.Save()

	case "unset":
		usage := "usage: plakar remote unset <name> [<key>, ...]"
		if len(args) == 0 {
			return fmt.Errorf(usage)
		}
		name := args[0]
		if !ctx.Config.HasRemote(name) {
			return fmt.Errorf("remote %q does not exists", name)
		}
		for _, key := range args[1:] {
			if key == "location" {
				return fmt.Errorf("cannot unset location")
			}
			delete(ctx.Config.Remotes[name], key)
		}
		return ctx.Config.Save()

	default:
		return fmt.Errorf(usage)
	}
}
