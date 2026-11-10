/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
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
	_ "embed"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	plogin "github.com/PlakarKorp/plakar/login"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Logout{} }, subcommands.BeforeRepositoryOpen, "logout")
}

func (cmd *Logout) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "logout",
	}
}

func (cmd *Logout) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) > 0 {
		return fmt.Errorf("too many arguments")
	}

	return nil
}

type Logout struct {
	subcommands.SubcommandBase
}

func (cmd *Logout) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	revokeErr, err := plogin.Logout(ctx)
	if revokeErr != nil {
		fmt.Fprintf(ctx.Stderr, "warning: the token was not revoked server-side: %v\n", revokeErr)
	}
	if err != nil {
		return 1, err
	}
	return 0, nil
}
