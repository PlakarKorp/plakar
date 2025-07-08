/*
 * Copyright (c) 2025 Eric Faurot <eric.faurot@plakar.io>
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
	"path/filepath"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &PkgLs{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "ls")
}

type PkgLs struct {
	subcommands.SubcommandBase
}

func (cmd *PkgLs) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg ls", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s",
			flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flag.PrintDefaults()
	}

	flags.Parse(args)

	if flags.NArg() != 0 {
		return fmt.Errorf("too many arguments")
	}

	return nil
}

func (cmd *PkgLs) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {

	dataDir, err := utils.GetDataDir("plakar")
	if err != nil {
		return 1, err
	}

	pluginsDir := filepath.Join(dataDir, "plugins")
	names, err := plugins.ListDir(ctx, pluginsDir)
	if err != nil {
		return 1, err
	}

	for _, name := range names {
		fmt.Fprintln(ctx.Stdout, name)
	}

	return 0, nil
}
