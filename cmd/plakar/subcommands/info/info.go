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

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
)

func init() {
	subcommands.Register("info", parse_cmd_info)
}

func parse_cmd_info(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	if len(args) == 0 {
		return &InfoRepository{
			RepositorySecret: ctx.GetSecret(),
		}, nil
	}

	flags := flag.NewFlagSet("info", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [SNAPSHOT]\n", flags.Name())
	}
	flags.Parse(args)

	if len(flags.Args()) > 1 {
		return nil, fmt.Errorf("invalid parameter. usage: info [snapshot]")
	}

	snapshotID, path := utils.ParseSnapshotID(flags.Arg(0))
	if snapshotID == "" {
		return nil, fmt.Errorf("invalid snapshot ID")
	}
	if path != "" {
		return &InfoVFS{
			RepositorySecret: ctx.GetSecret(),
			SnapshotPath:     flags.Arg(0),
		}, nil
	}

	return &InfoSnapshot{
		RepositorySecret: ctx.GetSecret(),
		SnapshotID:       flags.Args()[0],
	}, nil
}
