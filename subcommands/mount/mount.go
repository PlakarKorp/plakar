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

func init() {
	subcommands.Register(Mount, 0, "mount")
}

func Mount(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		Mountpoint    string
		LocateOptions *locate.LocateOptions
		AllowOthers   bool

		SnapshotPath string
	)

	LocateOptions = locate.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("mount", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [-to PATH] [snapshotID]\n", flags.Name())
	}
	flags.StringVar(&Mountpoint, "to", "", "mount point")
	flags.BoolVar(&AllowOthers, "allow-others", false, "allow other users to access the mount")
	LocateOptions.InstallLocateFlags(flags)
	flags.Parse(args)

	if flags.NArg() == 1 {
		// snapshot(s) level, reset LocateOptions
		LocateOptions = locate.NewDefaultLocateOptions()
		SnapshotPath = flags.Arg(0)
	}

	var chrootFS fs.FS

	if SnapshotPath != "" {
		snap, path, err := locate.OpenSnapshotByPath(repo, SnapshotPath)
		if err != nil {
			return err
		}

		pvfs, err := snap.Filesystem()
		if err != nil {
			return err
		}

		subFS, err := fs.Sub(pvfs, path[1:])
		if err != nil {
			return err
		}
		chrootFS = subFS
	}

	if strings.HasPrefix(Mountpoint, "http://") {
		return http.ExecuteHTTP(ctx, repo, Mountpoint, LocateOptions, chrootFS)
	}
	return fuse.ExecuteFUSE(ctx, repo, Mountpoint, LocateOptions, chrootFS, AllowOthers)
}
