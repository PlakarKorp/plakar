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

func init() {
	subcommands.MustRegister(func() subcommands.Subcommand { return &PkgAdd{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "add")

	subcommands.MustRegister(func() subcommands.Subcommand { return &PkgRm{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "rm")

	subcommands.MustRegister(func() subcommands.Subcommand { return &PkgCreate{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "create")

	subcommands.MustRegister(func() subcommands.Subcommand { return &PkgBuild{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "build")

	subcommands.MustRegister(func() subcommands.Subcommand { return &PkgList{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "list")
	subcommands.MustRegister(func() subcommands.Subcommand { return &PkgList{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "show")

	subcommands.MustRegister(func() subcommands.Subcommand { return &Pkg{} },
		subcommands.BeforeRepositoryOpen,
		"pkg")
}

type Pkg struct {
	subcommands.SubcommandBase
}

func (cmd *Pkg) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s list | add | build | create | rm\n",
			flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() > 0 {
		return fmt.Errorf("invalid argument: %s", flags.Arg(0))
	}
	return fmt.Errorf("no action specified")
}

func (cmd *Pkg) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	return 1, fmt.Errorf("no action specified")
}
