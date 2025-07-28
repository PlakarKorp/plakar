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
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
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
		//if !plugins.ValidateName(filepath.Base(name)) {
		//	return fmt.Errorf("bad plugin file name: %s", name)
		//}
		if !filepath.IsAbs(name) && !strings.HasPrefix(name, "./") {
			var recipe plugins.Recipe
			if err := plugins.GetRecipe(ctx, name, &recipe); err != nil {
				return fmt.Errorf("failed to parse the recipe %s: %w", flags.Arg(0), err)
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
	cachedir, err := utils.GetCacheDir("plakar")
	if err != nil {
		return 1, err
	}

	cachedir = filepath.Join(cachedir, "plugins", plugins.PLUGIN_API_VERSION)

	dataDir, err := utils.GetDataDir("plakar")
	if err != nil {
		return 1, err
	}

	pluginDir := filepath.Join(dataDir, "plugins", plugins.PLUGIN_API_VERSION)

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return 1, fmt.Errorf("failed to create plugin dir: %w", err)
	}

	for _, plugin := range cmd.Args {
		path, err := install(ctx, pluginDir, plugin)
		if err != nil {
			return 1, fmt.Errorf("failed to install %s: %w",
				filepath.Base(plugin), err)
		}

		err = plugins.Load(ctx, pluginDir, cachedir, filepath.Base(plugin))
		if err != nil {
			os.Remove(path)
			return 1, fmt.Errorf("failed to load %s: %w",
				filepath.Base(plugin), err)
		}
	}

	return 0, nil
}

func install(ctx *appcontext.AppContext, plugdir, plugin string) (string, error) {
	var name string
	var err error

	pluginName := strings.Split(filepath.Base(plugin), "_")[0]
	entries, err := os.ReadDir(plugdir)
	if err != nil {
		return "", fmt.Errorf("failed to read plugin dir %s: %w", plugdir, err)
	}
	for _, entry := range entries {
		if entry.Name() == filepath.Base(plugin) {
			return "", fmt.Errorf("plugin %s already installed", filepath.Base(plugin))
		}
		entryName := strings.Split(filepath.Base(entry.Name()), "_")[0]
		if entryName == pluginName {
			return "", fmt.Errorf("plugin already installed in a different version, remove first")
		}
	}

	if strings.HasPrefix(plugin, "https://") {
		u, err := url.Parse(plugin)
		if err != nil {
			return "", err
		}

		plugin, err = fetch(ctx, plugdir, plugin)
		if err != nil {
			return "", err
		}

		name = path.Base(u.Path)
		defer os.Remove(plugin)
	} else {
		name = filepath.Base(plugin)
	}

	dst := filepath.Join(plugdir, name)
	if err := os.Link(plugin, dst); err == nil {
		return dst, nil
	}

	fp, err := os.Open(plugin)
	if err != nil {
		return dst, err
	}
	defer fp.Close()

	// maybe a different filesystem
	tmp, err := os.CreateTemp(plugdir, "pkg-add-*")
	if err != nil {
		return dst, err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, fp); err != nil {
		return dst, err
	}

	return dst, os.Rename(tmp.Name(), dst)
}

func fetch(ctx *appcontext.AppContext, plugdir, plugin string) (string, error) {
	fp, err := os.CreateTemp(plugdir, "fetch-plugin-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer fp.Close()

	ctx.GetLogger().Info("fetching %s", plugin)
	req, err := http.Get(plugin)
	if err != nil {
		defer os.Remove(fp.Name())
		return "", fmt.Errorf("failed to fetch %s: %w", plugin, err)
	}
	defer req.Body.Close()

	if req.StatusCode/100 != 2 {
		defer os.Remove(fp.Name())

		if req.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("no package for %s/%s", runtime.GOOS, runtime.GOARCH)
		}

		return "", fmt.Errorf("failed to fetch %s: HTTP %d %s", plugin, req.StatusCode, req.Status)
	}

	if _, err := io.Copy(fp, req.Body); err != nil {
		defer os.Remove(fp.Name())
		return "", fmt.Errorf("failed to download the plugin: %w", err)
	}

	return fp.Name(), nil
}
