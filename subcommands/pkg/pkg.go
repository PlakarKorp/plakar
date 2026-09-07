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
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &PkgAdd{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "add")

	subcommands.Register(func() subcommands.Subcommand { return &PkgRm{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "rm")

	subcommands.Register(func() subcommands.Subcommand { return &PkgCreate{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "create")

	subcommands.Register(func() subcommands.Subcommand { return &PkgBuild{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "build")

	subcommands.Register(func() subcommands.Subcommand { return &PkgList{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "list")
	subcommands.Register(func() subcommands.Subcommand { return &PkgList{} },
		subcommands.BeforeRepositoryOpen,
		"pkg", "show")

	subcommands.Register(func() subcommands.Subcommand { return &Pkg{} },
		subcommands.BeforeRepositoryOpen,
		"pkg")
}

type Pkg struct {
	subcommands.SubcommandBase
}

func (cmd *Pkg) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "pkg",
	}
}

func (cmd *Pkg) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) > 0 {
		return fmt.Errorf("invalid argument: %s", rest[0])
	}
	return fmt.Errorf("no action specified")
}

func (cmd *Pkg) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	return 1, fmt.Errorf("no action specified")
}
