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

package login

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &TokenCreate{} }, subcommands.BeforeRepositoryOpen, "token", "create")
	subcommands.Register(func() subcommands.Subcommand { return &Token{} }, subcommands.BeforeRepositoryOpen, "token")
}

type Token struct {
	subcommands.SubcommandBase
}

func (_ *Token) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("token", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s create\n", flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() > 0 {
		return fmt.Errorf("Too many arguments")
	}

	return nil
}

func (cmd *Token) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	return 0, nil
}
