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
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.MustRegister(func() subcommands.Subcommand { return &Info{} }, subcommands.AgentSupport, "info")
}

type Info struct {
	subcommands.SubcommandBase
	SnapshotID string
	Errors     bool
}

func (cmd *Info) Parse(ctx *appcontext.AppContext, args []string) error {
	// Since this is the default action, we plug the general USAGE here.
	flags := flag.NewFlagSet("info", flag.ExitOnError)
	flags.BoolVar(&cmd.Errors, "errors", false, "display errors in the repository or snapshot")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [-errors] [SNAPSHOT]\n", flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() > 1 {
		return fmt.Errorf("too many arguments")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	if flags.NArg() == 1 {
		cmd.SnapshotID = flags.Arg(0)
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
