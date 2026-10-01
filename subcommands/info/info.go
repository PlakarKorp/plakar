/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
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

package info

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Info{} }, 0, "info")
}

type Info struct {
	subcommands.SubcommandBase
	SnapshotID string
	Errors     bool
}

func (cmd *Info) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "info [-errors] [SNAPSHOT]",
	}
	c.Flags().BoolVar(&cmd.Errors, "errors", false, "display errors in the repository or snapshot")
	return c
}

func (cmd *Info) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) > 1 {
		return fmt.Errorf("too many arguments")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	if len(rest) == 1 {
		cmd.SnapshotID = rest[0]
	}

	return nil
}

func (cmd *Info) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if cmd.SnapshotID == "" {
		return cmd.executeRepository(ctx, repo)
	}
	if cmd.Errors {
		return cmd.executeErrors(ctx, repo)
	}
	return cmd.executeSnapshot(ctx, repo)
}
