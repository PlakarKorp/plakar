/*
 * Copyright (c) 2025 Omar Polo <omar.polo@plakar.io>
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

package pkg

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/subcommands"
)

var baseURL, _ = url.Parse("https://plugins.plakar.io/kloset/pkg/" + plugins.PLUGIN_API_VERSION + "/")

type PkgAdd struct {
	subcommands.SubcommandBase
	Out      string
	Args     []string
	Manifest plugins.Manifest
}

func (cmd *PkgAdd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg add", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s plugin.ptar ...",
			flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flag.PrintDefaults()
	}

	flags.Parse(args)

	if flags.NArg() < 1 {
		return fmt.Errorf("not enough arguments")
	}

	cmd.Args = flags.Args()
	for i, name := range cmd.Args {
		if !filepath.IsAbs(name) && !strings.HasPrefix(name, "./") {
			var recipe plugins.Recipe
			if err := getRecipe(ctx, name, &recipe); err != nil {
				return fmt.Errorf("failed to parse the %q recipe: %w", name, err)
			}
			u := *baseURL
			u.Path = path.Join(u.Path, recipe.PkgName())
			name = u.String()
		} else if !filepath.IsAbs(name) {
			name = filepath.Join(ctx.CWD, name)
		}

		cmd.Args[i] = name
	}

	return nil
}

func (cmd *PkgAdd) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	for _, plugin := range cmd.Args {
		err := installPlugin(ctx, plugin)
		if err != nil {
			return 1, fmt.Errorf("failed to install %s: %w",
				filepath.Base(plugin), err)
		}
	}

	return 0, nil
}

func installPlugin(ctx *appcontext.AppContext, pluginFile string) error {
	var pkg plugins.Package

	err := plugins.ParsePackageName(path.Base(pluginFile), &pkg)
	if err != nil {
		return err
	}

	ok, _, err := ctx.GetPlugins().IsInstalled(pkg)
	if ok {
		return fmt.Errorf("package name %q already installed", pkg.Name)
	}

	if isRemote(pluginFile) {
		pluginFile, err = fetchPlugin(ctx, pluginFile)
		if err != nil {
			return err
		}
		defer os.Remove(pluginFile)
	}

	return ctx.GetPlugins().InstallPackage(ctx.GetInner(), pkg, pluginFile)
}

func fetchPlugin(ctx *appcontext.AppContext, path string) (string, error) {
	fp, err := os.CreateTemp(ctx.GetPlugins().PluginsDir, "fetch-plugin-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer fp.Close()

	rd, err := openURL(ctx, path)
	defer rd.Close()
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(fp, rd); err != nil {
		defer os.Remove(fp.Name())
		return "", fmt.Errorf("failed to download the plugin: %w", err)
	}

	return fp.Name(), nil
}
