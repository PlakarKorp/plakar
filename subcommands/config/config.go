/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"go.yaml.in/yaml/v3"
	"gopkg.in/ini.v1"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &ConfigStoreCmd{} },
		subcommands.BeforeRepositoryOpen, "store")
	subcommands.Register(func() subcommands.Subcommand { return &ConfigSourceCmd{} },
		subcommands.BeforeRepositoryOpen, "source")
	subcommands.Register(func() subcommands.Subcommand { return &ConfigDestinationCmd{} },
		subcommands.BeforeRepositoryOpen, "destination")
}

func normalizeLocation(location string) string {
	return strings.TrimPrefix(location, "location=")
}

func configure(ctx *appcontext.AppContext, cmd string, args []string) error {
	subcmd := "show"
	if len(args) > 0 {
		subcmd = args[0]
		args = args[1:]
	}

	err := dispatchSubcommand(ctx, cmd, subcmd, args)
	if err != nil {
		return err
	}
	return nil
}

func MarshalINISections(sectionName string, kv map[string]string, w io.Writer) error {
	cfg := ini.Empty()

	section := cfg.Section(sectionName)
	for key, value := range kv {
		section.Key(key).SetValue(value)
	}
	_, err := cfg.WriteTo(w)
	return err
}

func dispatchSubcommand(ctx *appcontext.AppContext, cmd string, subcmd string, args []string) error {
	var cfgMap map[string]map[string]string
	var hasFunc func(string) bool
	switch cmd {
	case "store":
		cfgMap = ctx.Config.Repositories
		hasFunc = ctx.Config.HasRepository
	case "destination":
		cfgMap = ctx.Config.Destinations
		hasFunc = ctx.Config.HasDestination
	case "source":
		cfgMap = ctx.Config.Sources
		hasFunc = ctx.Config.HasSource
	default:
		return fmt.Errorf("unknown cmd %q", cmd)
	}

	switch subcmd {
	case "add":
		if len(args) < 2 {
			return fmt.Errorf("usage: plakar %s %s <name> <location> [<key>=<value>, ...]", cmd, subcmd)
		}

		name, location := args[0], normalizeLocation(args[1])
		if hasFunc(name) {
			return fmt.Errorf("%s %q already exists", cmd, name)
		}
		cfgMap[name] = make(map[string]string)
		cfgMap[name]["location"] = location
		for _, kv := range args[2:] {
			key, val, found := strings.Cut(kv, "=")
			if !found || key == "" {
				return fmt.Errorf("usage: plakar %s %s <name> <location> [<key>=<value>, ...]", cmd, subcmd)
			}
			cfgMap[name][key] = val
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "check":
		if len(args) != 1 {
			return fmt.Errorf("usage: plakar %s check <name>", cmd)
		}
		name := args[0]
		if !hasFunc(name) {
			return fmt.Errorf("%s %q does not exists", cmd, name)
		}

		switch cmd {
		case "store":
			store, err := storage.New(ctx.GetInner(), cfgMap[name])
			if err != nil {
				return err
			}
			_ = store.Close()

		case "source":
			cfg, ok := ctx.Config.GetSource(name)
			if !ok {
				return fmt.Errorf("failed to retreive configuration for source %q", name)
			}
			imp, err := importer.NewImporter(ctx.GetInner(), ctx.ImporterOpts(), cfg)
			if err != nil {
				return err
			}
			_ = imp.Close()

		case "destination":
			cfg, ok := ctx.Config.GetDestination(name)
			if !ok {
				return fmt.Errorf("failed to retreive configuration for destination %q", name)
			}
			exp, err := exporter.NewExporter(ctx.GetInner(), cfg)
			if err != nil {
				return err
			}
			_ = exp.Close()
		}

		return nil

	case "import":
		if len(args) != 0 {
			return fmt.Errorf("usage: plakar %s import", cmd)
		}

		newConfMap, err := utils.GetConf(ctx.Stdin)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if len(newConfMap) == 0 {
			return fmt.Errorf("no valid %ss found in config", cmd)
		}
		for name, section := range newConfMap {
			if hasFunc(name) {
				fmt.Fprintf(ctx.Stderr, "%s %q already exists, skipping\n", cmd, name)
				continue
			}
			cfgMap[name] = make(map[string]string)
			maps.Copy(cfgMap[name], section)
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "ping":
		return fmt.Errorf("not implemented")

	case "rm":
		if len(args) != 1 {
			return fmt.Errorf("usage: plakar %s rm <name>", cmd)
		}
		name := args[0]
		if !hasFunc(name) {
			return fmt.Errorf("%s %q does not exist", cmd, name)
		}
		delete(cfgMap, name)
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "set":
		if len(args) == 0 {
			return fmt.Errorf("usage: plakar %s set <name> [<key>=<value>, ...]", cmd)
		}
		name := args[0]
		if !hasFunc(name) {
			return fmt.Errorf("%s %q does not exists", cmd, name)
		}
		for _, kv := range args[1:] {
			key, val, found := strings.Cut(kv, "=")
			if !found || key == "" {
				return fmt.Errorf("usage: plakar %s set <name> [<key>=<value>, ...]", cmd)
			}
			cfgMap[name][key] = val
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	case "show":
		var opt_json bool
		var opt_ini bool
		var opt_yaml bool
		p := flag.NewFlagSet("show", flag.ExitOnError)
		p.BoolVar(&opt_json, "json", false, "output in JSON format")
		p.BoolVar(&opt_ini, "ini", false, "output in INI format")
		p.BoolVar(&opt_yaml, "yaml", false, "output in YAML format (default)")
		p.Parse(args)

		names := make([]string, 0)
		if len(args) == 0 {
			for name := range cfgMap {
				names = append(names, name)
			}
		} else {
			names = p.Args()
		}
		for _, name := range names {
			if !hasFunc(name) {
				fmt.Fprintf(ctx.Stderr, "%s %q does not exist\n", cmd, name)
				continue
			}

			var err error
			if opt_json {
				err = json.NewEncoder(ctx.Stdout).Encode(map[string]map[string]string{name: cfgMap[name]})
			} else if opt_ini {
				err = MarshalINISections(name, cfgMap[name], ctx.Stdout)
			} else {
				err = yaml.NewEncoder(ctx.Stdout).Encode(map[string]map[string]string{name: cfgMap[name]})
			}
			if err != nil {
				return fmt.Errorf("failed to encode store %q: %w", name, err)
			}
		}
		return nil

	case "unset":
		if len(args) == 0 {
			return fmt.Errorf("usage: plakar %s unset <name> [<key>, ...]", cmd)
		}
		name := args[0]
		if !hasFunc(name) {
			return fmt.Errorf("%s %q does not exists", cmd, name)
		}
		for _, key := range args[1:] {
			if key == "location" {
				return fmt.Errorf("cannot unset location")
			}
			delete(cfgMap[name], key)
		}
		return utils.SaveConfig(ctx.ConfigDir, ctx.Config)

	default:
		return fmt.Errorf("usage: plakar %s [add|check|import|ls|ping|rm|set|show|unset]", cmd)
	}
}
