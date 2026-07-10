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
	"io/fs"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/mount/fuse"
	"github.com/PlakarKorp/plakar/subcommands/mount/http"
)

type Mount struct {
	subcommands.SubcommandBase

	Mountpoint    string
	LocateOptions *locate.LocateOptions
	AllowOthers   bool

	SnapshotPath string
}

const snapshotRootPath = "/"

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Mount{} }, 0, "mount")
}

func (cmd *Mount) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.LocateOptions = locate.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("mount", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [-to PATH] [snapshotID]\n", flags.Name())
	}
	flags.StringVar(&cmd.Mountpoint, "to", "", "mount point")
	flags.BoolVar(&cmd.AllowOthers, "allow-others", false, "allow other users to access the mount")
	cmd.LocateOptions.InstallLocateFlags(flags)
	flags.Parse(args)

	cmd.RepositorySecret = ctx.GetSecret()

	if flags.NArg() == 1 {
		// snapshot(s) level, reset LocateOptions
		cmd.LocateOptions = locate.NewDefaultLocateOptions()
		cmd.SnapshotPath = flags.Arg(0)
	}

	return nil
}

func (cmd *Mount) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var chrootFS fs.FS

	if cmd.SnapshotPath != "" {
		snap, path, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPath)
		if err != nil {
			return 1, err
		}

		pvfs, err := snap.Filesystem()
		if err != nil {
			return 1, err
		}

		subFS, err := snapshotSubFS(pvfs, path)
		if err != nil {
			return 1, err
		}
		chrootFS = subFS
	}

	if strings.HasPrefix(cmd.Mountpoint, "http://") {
		return http.ExecuteHTTP(ctx, repo, cmd.Mountpoint, cmd.LocateOptions, chrootFS)
	}
	return fuse.ExecuteFUSE(ctx, repo, cmd.Mountpoint, cmd.LocateOptions, chrootFS, cmd.AllowOthers)
}

func snapshotSubFS(snapshotFS fs.FS, snapshotPath string) (fs.FS, error) {
	if snapshotPath == snapshotRootPath {
		return snapshotFS, nil
	}
	return fs.Sub(snapshotFS, strings.TrimPrefix(snapshotPath, snapshotRootPath))
}
