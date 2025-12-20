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

package check

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/google/uuid"
)

type Check struct {
	subcommands.SubcommandBase

	LocateOptions *locate.LocateOptions
	FastCheck     bool
	NoVerify      bool
	Snapshots     []string
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Check{} }, 0, "check")
}

func (cmd *Check) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.LocateOptions = locate.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("check", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&cmd.NoVerify, "no-verify", false, "disable signature verification")
	flags.BoolVar(&cmd.FastCheck, "fast", false, "enable fast checking (no digest verification)")
	cmd.LocateOptions.InstallLocateFlags(flags)

	flags.Parse(args)

	if flags.NArg() != 0 && !cmd.LocateOptions.Empty() {
		ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Snapshots = flags.Args()

	return nil
}

func (cmd *Check) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var snapshots []string
	if len(cmd.Snapshots) == 0 {
		snapshotIDs, err := locate.LocateSnapshotIDs(repo, cmd.LocateOptions)
		if err != nil {
			return 1, err
		}
		for _, snapshotID := range snapshotIDs {
			snapshots = append(snapshots, fmt.Sprintf("%x:", snapshotID))
		}
	} else {
		for _, snapshotPath := range cmd.Snapshots {
			prefix, path := locate.ParseSnapshotPath(snapshotPath)
			if prefix != "" {
				if _, err := hex.DecodeString(prefix); err != nil {
					return 1, fmt.Errorf("invalid snapshot prefix: %s", prefix)
				}
			}

			cmd.LocateOptions.Filters.IDs = []string{prefix}
			snapshotIDs, err := locate.LocateSnapshotIDs(repo, cmd.LocateOptions)
			if err != nil {
				fmt.Fprintln(ctx.Stderr, err)
				return 1, err
			}

			for _, snapshotID := range snapshotIDs {
				snapshots = append(snapshots, fmt.Sprintf("%x:%s", snapshotID, path))
			}
		}
	}

	opts := &snapshot.CheckOptions{
		FastCheck: cmd.FastCheck,
	}

	checkCache, err := ctx.GetCache().Check()
	if err != nil {
		return 1, err
	}
	defer checkCache.Close()

	var failures int
	for _, arg := range snapshots {
		snap, pathname, err := locate.OpenSnapshotByPath(repo, arg)
		if err != nil {
			return 1, err
		}

		snap.SetCheckCache(checkCache)

		var failed bool
		if !cmd.NoVerify && snap.Header.Identity.Identifier != uuid.Nil {
			if ok, err := snap.Verify(); err != nil {
				ctx.GetLogger().Warn("%s", err)
			} else if !ok {
				ctx.GetLogger().Info("snapshot %x signature verification failed", snap.Header.Identifier)
				failed = true
			} else {
				ctx.GetLogger().Info("snapshot %x signature verification succeeded", snap.Header.Identifier)
			}
		}

		if err := snap.Check(pathname, opts); err != nil {
			ctx.GetLogger().Warn("check failed for snapshot %x: %s",
				snap.Header.GetIndexID(), err)
			failed = true
		}

		if failed {
			failures++
		} else {
			ctx.GetLogger().Info("check: verification of %x:%s completed successfully",
				snap.Header.GetIndexShortID(),
				pathname)
		}

		snap.Close()
	}

	if failures != 0 {
		snapshots := "snapshots"
		if failures == 1 {
			snapshots = "snapshot"
		}
		return 1, fmt.Errorf("check failed for %d %s",
			failures, snapshots)
	}

	return 0, nil
}
