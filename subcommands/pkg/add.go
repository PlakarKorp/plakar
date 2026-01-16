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
	"os"
	"path/filepath"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type PkgAdd struct {
	subcommands.SubcommandBase
	Args []string
}

func (cmd *PkgAdd) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg add", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), `Usage: %s <package> ...

Arguments:
  <package>    Local .ptar file, or recipe name to fetch from plugins.plakar.io
               (local files take precedence over remote recipes)

Examples:
  pkg add imap           Fetch and install the 'imap' plugin
  pkg add ./plugin.ptar  Install from local file
`, flags.Name())
	}

	flags.Parse(args)

	if flags.NArg() < 1 {
		return fmt.Errorf("not enough arguments")
	}

	cmd.Args = flags.Args()
	for i, name := range cmd.Args {
		absolute := name
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(ctx.CWD, absolute)
		}

		if info, err := os.Stat(absolute); err == nil && info.Mode().IsRegular() {
			cmd.Args[i] = absolute
			continue
		}

		if filepath.IsAbs(name) {
			return fmt.Errorf("file not found: %s", name)
		}
	}

	return nil
}

func (cmd *PkgAdd) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	addopts := pkg.AddOptions{
		ImplicitFetch: true,
	}

	pkgmgr := ctx.GetPkgManager()
	for _, plugin := range cmd.Args {
		if err := pkgmgr.Add(plugin, &addopts); err != nil {
			return 1, fmt.Errorf("failed to install %s: %w",
				filepath.Base(plugin), err)
		}
	}

	return 0, nil
}
