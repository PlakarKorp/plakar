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

package mount

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Mount{} }, subcommands.AgentSupport, "mount")
}

func (cmd *Mount) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.LocateOptions = locate.NewDefaultLocateOptions()
	flags := flag.NewFlagSet("mount", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s PATH\n", flags.Name())
	}
	flags.StringVar(&cmd.Mountpoint, "to", "", "Mountpoint to use for the FUSE filesystem")
	cmd.LocateOptions.InstallLocateFlags(flags)
	flags.Parse(args)

	if cmd.Mountpoint == "" {
		return fmt.Errorf("option -to is required to specify a mountpoint")
	}

	if flags.NArg() != 0 && !cmd.LocateOptions.Empty() {
		ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Snapshots = flags.Args()

	return nil
}

type Mount struct {
	subcommands.SubcommandBase

	LocateOptions *locate.LocateOptions
	Mountpoint    string
	Snapshots     []string
}
