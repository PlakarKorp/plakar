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

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type PkgRm struct {
	subcommands.SubcommandBase
	Args []string
}

func (cmd *PkgRm) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg rm", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s plugins...\n",
			flags.Name())
	}

	flags.Parse(args)

	cmd.Args = flags.Args()

	return nil
}

func (cmd *PkgRm) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	pkgmgr := ctx.GetPkgManager()
	for _, name := range cmd.Args {
		if err := pkgmgr.Del(name, nil); err != nil {
			return 1, fmt.Errorf("failed to remove %q: %w", name, err)
		}
	}

	return 0, nil
}
