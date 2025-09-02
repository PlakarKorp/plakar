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

package digest

import (
	"flag"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Digest{} }, subcommands.AgentSupport, "digest")
}

func (cmd *Digest) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_hashing string

	flags := flag.NewFlagSet("digest", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&opt_hashing, "hashing", "SHA256", "hashing algorithm to use")
	flags.Parse(args)

	if flags.NArg() == 0 {
		return fmt.Errorf("at least one parameter is required")
	}

	hashingFunction := strings.ToUpper(opt_hashing)
	if hashing.GetHasher(hashingFunction) == nil {
		return fmt.Errorf("unsupported hashing algorithm: %s", hashingFunction)
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.HashingFunction = hashingFunction
	cmd.Targets = flags.Args()

	return nil
}

type Digest struct {
	subcommands.SubcommandBase

	HashingFunction string
	Targets         []string
}

func (cmd *Digest) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	errors := 0
	for _, snapshotPath := range cmd.Targets {
		snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath)
		if err != nil {
			ctx.GetLogger().Error("digest: %s: %s", pathname, err)
			errors++
			continue
		}

		fs, err := snap.Filesystem()
		if err != nil {
			snap.Close()
			continue
		}

		cmd.displayDigests(ctx, fs, repo, snap, pathname)
		snap.Close()
	}

	return 0, nil
}

func (cmd *Digest) displayDigests(ctx *appcontext.AppContext, fs *vfs.Filesystem, repo *repository.Repository, snap *snapshot.Snapshot, pathname string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	fsinfo, err := fs.GetEntry(pathname)
	if err != nil {
		return err
	}

	if fsinfo.Stat().Mode().IsDir() {
		iter, err := fsinfo.Getdents(fs)
		if err != nil {
			return err
		}
		for child := range iter {
			if err := cmd.displayDigests(ctx, fs, repo, snap, path.Join(pathname, child.Stat().Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !fsinfo.Stat().Mode().IsRegular() {
		return nil
	}

	rd, err := snap.NewReader(pathname)
	if err != nil {
		return err
	}
	defer rd.Close()

	algorithm := cmd.HashingFunction
	hasher := hashing.GetHasher(algorithm)
	if _, err := io.Copy(hasher, rd); err != nil {
		return err
	}
	digest := hasher.Sum(nil)
	fmt.Fprintf(ctx.Stdout, "%s (%s) = %x\n", algorithm, utils.SanitizeText(pathname), digest)
	return nil
}
