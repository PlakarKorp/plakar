//go:build linux || darwin
// +build linux darwin

package mount

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

import (
	"fmt"
	"os"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plakarfs"
	"github.com/anacrolix/fuse"
	"github.com/anacrolix/fuse/fs"
)

func (cmd *Mount) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if _, err := os.Stat(cmd.Mountpoint); err != nil {
		if !os.IsNotExist(err) {
			return 1, fmt.Errorf("mount: %v", err)
		}
		if err := os.MkdirAll(cmd.Mountpoint, 0700); err != nil {
			return 1, fmt.Errorf("mount: cannot create mountpoint %s: %v", cmd.Mountpoint, err)
		}
	}

	location, err := repo.Location()
	if err != nil {
		return 1, fmt.Errorf("mount: cannot get repository location: %v", err)
	}

	c, err := fuse.Mount(
		cmd.Mountpoint,
		fuse.FSName("plakar"),
		fuse.Subtype("plakarfs"),
		fuse.LocalVolume(),
	)

	if err != nil {
		return 1, fmt.Errorf("mount: %v", err)
	}

	return cmd.RunForeground(ctx, repo, location, c)
}

func (cmd *Mount) RunForeground(ctx *appcontext.AppContext, repo *repository.Repository, location string, c *fuse.Conn) (int, error) {
	done := make(chan error, 1)

	go func() {
		done <- fs.Serve(c, plakarfs.NewFS(repo, cmd.LocateOptions, cmd.Snapshots, cmd.Mountpoint))
	}()

	<-c.Ready
	if err := c.MountError; err != nil {
		c.Close()
		return 1, fmt.Errorf("mount error: %v", err)
	}

	ctx.GetLogger().Info("mounted %s at %s", location, cmd.Mountpoint)

	go func() {
		<-ctx.Done()
		if err := fuse.Unmount(cmd.Mountpoint); err != nil {
			ctx.GetLogger().Error("unmount failed: %v", err)
		}
	}()

	err := <-done
	defer c.Close()

	if err != nil {
		return 1, fmt.Errorf("mount: %v", err)
	}
	return 0, nil
}
