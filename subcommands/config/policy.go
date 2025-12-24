package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

type ConfigPolicyCmd struct {
	subcommands.SubcommandBase

	args []string
}

func (cmd *ConfigPolicyCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("policy", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s add <name> [<key>=<value>]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s rm <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s set <name> [<option>=<value>...]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s show [<name>...]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s unset <name> <option>...\n", flags.Name())
		flags.PrintDefaults()
	}

	flags.Parse(args)
	if flags.NArg() == 0 {
		return fmt.Errorf("no action specified")
	}
	cmd.args = flags.Args()
	return nil
}

func (cmd *ConfigPolicyCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	err := dispatchPolicy(ctx, "policy", cmd.args[0], cmd.args[1:])
	if err != nil {
		return 1, err
	}
	return 0, nil
}

func dispatchPolicy(ctx *appcontext.AppContext, cmd, subcmd string, args []string) error {
	configFile := filepath.Join(ctx.ConfigDir, "policies.yml")
	config, err := utils.LoadPolicyConfigFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	switch subcmd {
	case "add":
		p := flag.NewFlagSet("add", flag.ExitOnError)
		p.Usage = func() {
			fmt.Printf("Usage: plakar %s %s <name> [<key>=<value>...]\n", cmd, p.Name())
			p.PrintDefaults()
		}
		p.Parse(args)

		if len(args) < 1 {
			return fmt.Errorf("Usage: plakar %s %s <name> [<key>=<value>...]", cmd, p.Name())
		}

		name := normalizeName(args[0])
		if config.Has(name) {
			return fmt.Errorf("%s %q already exists", cmd, name)
		}
		config.Add(name)
		for _, kv := range args[1:] {
			key, val, found := strings.Cut(kv, "=")
			if !found || key == "" {
				return fmt.Errorf("Usage: plakar %s %s <name> [<key>=<value>...]", cmd, p.Name())
			}
			if err := config.Set(name, key, val); err != nil {
				return fmt.Errorf("failed to set key %q: %w", key, err)
			}
		}
		return config.SaveToFile(configFile)

	case "rm":
		p := flag.NewFlagSet("rm", flag.ExitOnError)
		p.Usage = func() {
			fmt.Printf("Usage: plakar %s %s <name>\n", cmd, p.Name())
			p.PrintDefaults()
		}
		p.Parse(args)

		if len(args) != 1 {
			return fmt.Errorf("Usage: plakar %s %s <name>", cmd, p.Name())
		}

		name := normalizeName(args[0])
		if !config.Has(name) {
			return fmt.Errorf("%s %q does not exist", cmd, name)
		}
		config.Remove(name)
		return config.SaveToFile(configFile)

	case "set":
		p := flag.NewFlagSet("set", flag.ExitOnError)
		p.Usage = func() {
			fmt.Printf("Usage: plakar %s %s <name> <key>=<value>...\n", cmd, p.Name())
			p.PrintDefaults()
		}
		p.Parse(args)

		if len(args) < 2 {
			return fmt.Errorf("Usage: plakar %s %s <name> <key>=<value>...", cmd, p.Name())
		}
		name := normalizeName(args[0])
		if !config.Has(name) {
			return fmt.Errorf("%s %q does not exists", cmd, name)
		}
		for _, kv := range args[1:] {
			key, val, found := strings.Cut(kv, "=")
			if !found || key == "" {
				return fmt.Errorf("usage: plakar %s set <name> [<key>=<value>, ...]", cmd)
			}
			if err := config.Set(name, key, val); err != nil {
				return fmt.Errorf("failed to set key %q: %w", key, err)
			}
		}
		return config.SaveToFile(configFile)

	case "show":
		var opt_json bool
		var opt_ini bool
		var opt_yaml bool
		p := flag.NewFlagSet("show", flag.ExitOnError)
		p.Usage = func() {
			fmt.Printf("Usage: plakar %s %s [<name>...]\n", cmd, p.Name())
			p.PrintDefaults()
		}

		p.BoolVar(&opt_json, "json", false, "output in JSON format")
		p.BoolVar(&opt_ini, "ini", false, "output in INI format")
		p.BoolVar(&opt_yaml, "yaml", false, "output in YAML format (default)")
		p.Parse(args)

		names := make([]string, 0)
		for _, name := range p.Args() {
			names = append(names, normalizeName(name))
		}

		format := "yaml"
		if opt_json {
			format = "json"
		} else if opt_ini {
			format = "ini"
		}

		return config.Dump(os.Stdout, format, names)

	case "unset":
		p := flag.NewFlagSet("unset", flag.ExitOnError)
		p.Usage = func() {
			fmt.Printf("Usage: plakar %s %s <name> <key>...\n", cmd, p.Name())
			p.PrintDefaults()
		}
		p.Parse(args)

		if len(args) < 2 {
			return fmt.Errorf("Usage: plakar %s %s <name> <key>...", cmd, p.Name())
		}
		name := normalizeName(args[0])
		if !config.Has(name) {
			return fmt.Errorf("%s %q does not exists", cmd, name)
		}
		for _, key := range args[1:] {
			config.Unset(name, key)
		}
		return config.SaveToFile(configFile)

	default:
		return fmt.Errorf("usage: plakar %s [add|rm|set|show|unset]", cmd)
	}
}
