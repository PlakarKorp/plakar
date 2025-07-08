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
	"os"
	"path/filepath"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &PkgInstall{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "install")
}

type PkgInstall struct {
	subcommands.SubcommandBase
	Out      string
	Args     []string
	Manifest plugins.Manifest
}

func (cmd *PkgInstall) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg install", flag.ExitOnError)
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
		if !plugins.ValidateName(filepath.Base(name)) {
			return fmt.Errorf("bad plugin file name: %s", name)
		}
		if !filepath.IsAbs(name) {
			cmd.Args[i] = filepath.Join(ctx.CWD, name)
		}
	}

	return nil
}

func (cmd *PkgInstall) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	cachedir, err := utils.GetCacheDir("plakar")
	if err != nil {
		return 1, err
	}

	cachedir = filepath.Join(cachedir, "plugins")

	dataDir, err := utils.GetDataDir("plakar")
	if err != nil {
		return 1, err
	}

	pluginDir := filepath.Join(dataDir, "plugins")

	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return 1, fmt.Errorf("failed to create plugin dir: %w", err)
	}

	for _, plugin := range cmd.Args {
		path, err := install(pluginDir, plugin)
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

func install(plugdir, plugin string) (string, error) {
	dst := filepath.Join(plugdir, filepath.Base(plugin))
	if err := os.Link(plugin, dst); err == nil {
		return dst, nil
	}

	fp, err := os.Open(plugin)
	if err != nil {
		return dst, err
	}
	defer fp.Close()

	// maybe a different filesystem
	tmp, err := os.CreateTemp(plugdir, "pkg-install-*")
	if err != nil {
		return dst, err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, fp); err != nil {
		return dst, err
	}

	return dst, os.Rename(tmp.Name(), dst)
}
