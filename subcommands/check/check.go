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
	"github.com/PlakarKorp/plakar/exitcodes"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/google/uuid"
)

func init() {
	subcommands.Register(Check, 0, "check")
}

func Check(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		LocateOptions = locate.NewDefaultLocateOptions()
		FastCheck     bool
		NoVerify      bool
		Snapshots     []string
	)

	flags := flag.NewFlagSet("check", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&NoVerify, "no-verify", false, "disable signature verification")
	flags.BoolVar(&FastCheck, "fast", false, "enable fast checking (no digest verification)")
	LocateOptions.InstallLocateFlags(flags)

	flags.Parse(args)

	if flags.NArg() != 0 && !LocateOptions.Empty() {
		ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
	}

	var snapshots []string
	if len(Snapshots) == 0 {
		snapshotIDs, err := locate.LocateSnapshotIDs(repo, LocateOptions)
		if err != nil {
			return err
		}
		for _, snapshotID := range snapshotIDs {
			snapshots = append(snapshots, fmt.Sprintf("%x:", snapshotID))
		}
	} else {
		for _, snapshotPath := range Snapshots {
			prefix, path := locate.ParseSnapshotPath(snapshotPath)
			if prefix != "" {
				if _, err := hex.DecodeString(prefix); err != nil {
					return fmt.Errorf("invalid snapshot prefix: %s", prefix)
				}
			}

			LocateOptions.Filters.IDs = []string{prefix}
			snapshotIDs, err := locate.LocateSnapshotIDs(repo, LocateOptions)
			if err != nil {
				return err
			}

			for _, snapshotID := range snapshotIDs {
				snapshots = append(snapshots, fmt.Sprintf("%x:%s", snapshotID, path))
			}
		}
	}

	opts := &snapshot.CheckOptions{
		FastCheck: FastCheck,
	}

	checkCache, err := ctx.GetCache().Check()
	if err != nil {
		return err
	}
	defer checkCache.Close()

	emitter := repo.Emitter("check")
	defer emitter.Close()

	var failures int
	for _, arg := range snapshots {
		snap, pathname, err := locate.OpenSnapshotByPath(repo, arg)
		if err != nil {
			return err
		}

		snap.SetCheckCache(checkCache)

		var failed bool
		if !NoVerify && snap.Header.Identity.Identifier != uuid.Nil {
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
		return subcommands.NewErrCode(exitcodes.IntegrityFailure, "check failed for %d %s",
			failures, snapshots)
	}

	return nil
}
