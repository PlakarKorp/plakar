//go:build linux || darwin
// +build linux darwin

package fuse

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
	"io/fs"
	"os"
	"syscall"

	"path/filepath"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands/mount/fuse/plakarfs"
	"github.com/anacrolix/fuse"
	fusefs "github.com/anacrolix/fuse/fs"
	"github.com/google/uuid"
)

func ExecuteFUSE(ctx *appcontext.AppContext, repo *repository.Repository, mountpoint string, locateOptions *locate.LocateOptions, chrootfs fs.FS) (int, error) {
	if mountpoint == "" {
		mountpoint = filepath.Join(ctx.CWD, uuid.New().String())
		if err := os.MkdirAll(mountpoint, 0700); err != nil {
			return 1, err
		}
		defer os.Remove(mountpoint)
	} else {
		mp, err := looksLikeMountpoint(mountpoint)
		if err != nil {
			return 1, err
		}
		if mp {
			return 1, fmt.Errorf("%s already looks like a mountpoint; refusing to mount over it", mountpoint)
		}
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("plakar"),
		fuse.Subtype("plakarfs"),
		fuse.LocalVolume(),
		fuse.ReadOnly(),
	)
	if err != nil {
		return 1, fmt.Errorf("mount: %v", err)
	}
	defer c.Close()

	// fuse.Mount returns as soon as the kernel hands back a connection;
	// the actual mount succeeds asynchronously and is signaled via c.Ready.
	// Check for early mount errors before entering Serve, which blocks
	// until unmount.
	select {
	case <-c.Ready:
		if err := c.MountError; err != nil {
			return 1, fmt.Errorf("mount: %v", err)
		}
	default:
	}

	ctx.GetLogger().Info("mounted repository %s at %s", repo.Origin(), mountpoint)

	go func() {
		<-ctx.Done()

		if err := fuse.Unmount(mountpoint); err != nil {
			fmt.Fprintf(os.Stderr, "%s is still in use; run `umount -f %s` to force\n", mountpoint, mountpoint)
		}
	}()

	if err := fusefs.Serve(c, plakarfs.NewFS(ctx, repo, locateOptions, chrootfs)); err != nil {
		return 1, err
	}

	<-c.Ready
	if err := c.MountError; err != nil {
		return 1, err
	}
	return 0, nil
}

func looksLikeMountpoint(p string) (bool, error) {
	p = filepath.Clean(p)

	parent := filepath.Dir(p)
	if parent == p {
		return true, nil
	}

	var stP, stParent syscall.Stat_t
	if err := syscall.Lstat(p, &stP); err != nil {
		return false, err
	}
	if err := syscall.Lstat(parent, &stParent); err != nil {
		return false, err
	}

	if stP.Dev != stParent.Dev {
		return true, nil
	}

	// could still be a bind mount; we can’t detect that portably.
	return false, nil
}
