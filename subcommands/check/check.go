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
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/exitcodes"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
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

func (cmd *Check) CobraCommand() *cobra.Command {
	cmd.LocateOptions = locate.NewDefaultLocateOptions()

	c := &cobra.Command{
		Use: "check [OPTIONS] [SNAPSHOT[:PATH]]...",
	}
	c.Flags().BoolVar(&cmd.NoVerify, "no-verify", false, "disable signature verification")
	c.Flags().BoolVar(&cmd.FastCheck, "fast", false, "enable fast checking (no digest verification)")
	subcommands.InstallGoFlags(c.Flags(), cmd.LocateOptions.InstallLocateFlags)
	return c
}

func (cmd *Check) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) != 0 && !cmd.LocateOptions.Empty() {
		ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Snapshots = rest

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

	// Each snapshot's Check (and Verify) emits its own "check" workflow, so the
	// UI shows one progress view per snapshot — consistent with sync/backup.
	// A run-spanning emitter here would collapse them into a single bar.
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
			failed = true
		}

		if failed {
			failures++
		}

		snap.Close()
	}

	if failures != 0 {
		snapshots := "snapshots"
		if failures == 1 {
			snapshots = "snapshot"
		}
		return exitcodes.IntegrityFailure, fmt.Errorf("check failed for %d %s",
			failures, snapshots)
	}

	return 0, nil
}
