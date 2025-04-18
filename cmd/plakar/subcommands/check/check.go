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
	"flag"
	"fmt"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/google/uuid"
)

func init() {
	subcommands.Register(&Check{}, "check")
}

func (cmd *Check) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_before string
	var opt_since string

	flags := flag.NewFlagSet("check", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.Uint64Var(&cmd.Concurrency, "concurrency", uint64(ctx.MaxConcurrency), "maximum number of parallel tasks")
	flags.StringVar(&cmd.OptName, "name", "", "filter by name")
	flags.StringVar(&cmd.OptCategory, "category", "", "filter by category")
	flags.StringVar(&cmd.OptEnvironment, "environment", "", "filter by environment")
	flags.StringVar(&cmd.OptPerimeter, "perimeter", "", "filter by perimeter")
	flags.StringVar(&cmd.OptJob, "job", "", "filter by job")
	flags.StringVar(&cmd.OptTag, "tag", "", "filter by tag")
	flags.StringVar(&opt_before, "before", "", "filter by date")
	flags.StringVar(&opt_since, "since", "", "filter by date")
	flags.BoolVar(&cmd.OptLatest, "latest", false, "use latest snapshot")
	flags.BoolVar(&cmd.NoVerify, "no-verify", false, "disable signature verification")
	flags.BoolVar(&cmd.FastCheck, "fast", false, "enable fast checking (no digest verification)")
	flags.BoolVar(&cmd.Quiet, "quiet", false, "suppress output")
	flags.BoolVar(&cmd.Silent, "silent", false, "suppress ALL output")
	flags.Parse(args)

	var err error

	var beforeDate time.Time
	if opt_before != "" {
		beforeDate, err = utils.ParseTimeFlag(opt_before)
		if err != nil {
			return fmt.Errorf("invalid date format: %s", opt_before)
		}

		cmd.OptBefore = beforeDate
	}

	var sinceDate time.Time
	if opt_since != "" {
		sinceDate, err = utils.ParseTimeFlag(opt_since)
		if err != nil {
			return fmt.Errorf("invalid date format: %s", opt_since)
		}

		cmd.OptSince = sinceDate
	}

	if flags.NArg() != 0 {
		if cmd.OptName != "" || cmd.OptCategory != "" || cmd.OptEnvironment != "" || cmd.OptPerimeter != "" || cmd.OptJob != "" || cmd.OptTag != "" || !beforeDate.IsZero() || !sinceDate.IsZero() || cmd.OptLatest {
			ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
		}
	}

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

type Check struct {
	subcommands.SubcommandBase

	OptBefore time.Time
	OptSince  time.Time
	OptLatest bool

	OptName        string
	OptCategory    string
	OptEnvironment string
	OptPerimeter   string
	OptJob         string
	OptTag         string

	Concurrency uint64
	FastCheck   bool
	NoVerify    bool
	Quiet       bool
	Snapshots   []string
	Silent      bool
}

func (cmd *Check) Name() string {
	return "check"
}

func (cmd *Check) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if !cmd.Silent {
		go eventsProcessorStdio(ctx, cmd.Quiet)
	}

	var snapshots []string
	if len(cmd.Snapshots) == 0 {
		locateOptions := utils.NewDefaultLocateOptions()
		locateOptions.MaxConcurrency = ctx.MaxConcurrency
		locateOptions.SortOrder = utils.LocateSortOrderAscending

		locateOptions.Before = cmd.OptBefore
		locateOptions.Since = cmd.OptSince
		locateOptions.Latest = cmd.OptLatest

		locateOptions.Name = cmd.OptName
		locateOptions.Category = cmd.OptCategory
		locateOptions.Environment = cmd.OptEnvironment
		locateOptions.Perimeter = cmd.OptPerimeter
		locateOptions.Job = cmd.OptJob
		locateOptions.Tag = cmd.OptTag

		snapshotIDs, err := utils.LocateSnapshotIDs(repo, locateOptions)
		if err != nil {
			return 1, err
		}
		for _, snapshotID := range snapshotIDs {
			snapshots = append(snapshots, fmt.Sprintf("%x:", snapshotID))
		}
	} else {
		for _, snapshotPath := range cmd.Snapshots {
			prefix, path := utils.ParseSnapshotPath(snapshotPath)

			locateOptions := utils.NewDefaultLocateOptions()
			locateOptions.MaxConcurrency = ctx.MaxConcurrency
			locateOptions.SortOrder = utils.LocateSortOrderAscending

			locateOptions.Before = cmd.OptBefore
			locateOptions.Since = cmd.OptSince
			locateOptions.Latest = cmd.OptLatest

			locateOptions.Name = cmd.OptName
			locateOptions.Category = cmd.OptCategory
			locateOptions.Environment = cmd.OptEnvironment
			locateOptions.Perimeter = cmd.OptPerimeter
			locateOptions.Job = cmd.OptJob
			locateOptions.Tag = cmd.OptTag
			locateOptions.Prefix = prefix

			snapshotIDs, err := utils.LocateSnapshotIDs(repo, locateOptions)
			if err != nil {
				return 1, err
			}
			for _, snapshotID := range snapshotIDs {
				snapshots = append(snapshots, fmt.Sprintf("%x:%s", snapshotID, path))
			}
		}
	}

	opts := &snapshot.CheckOptions{
		MaxConcurrency: cmd.Concurrency,
		FastCheck:      cmd.FastCheck,
	}

	checkCache, err := ctx.GetCache().Check()
	if err != nil {
		return 1, err
	}
	defer checkCache.Close()

	failures := false
	for _, arg := range snapshots {
		snap, pathname, err := utils.OpenSnapshotByPath(repo, arg)
		if err != nil {
			return 1, err
		}

		snap.SetCheckCache(checkCache)

		if !cmd.NoVerify && snap.Header.Identity.Identifier != uuid.Nil {
			if ok, err := snap.Verify(); err != nil {
				ctx.GetLogger().Warn("%s", err)
			} else if !ok {
				ctx.GetLogger().Info("snapshot %x signature verification failed", snap.Header.Identifier)
				failures = true
			} else {
				ctx.GetLogger().Info("snapshot %x signature verification succeeded", snap.Header.Identifier)
			}
		}

		if ok, err := snap.Check(pathname, opts); err != nil {
			ctx.GetLogger().Warn("%s", err)
		} else if !ok {
			failures = true
		}

		if !failures {
			ctx.GetLogger().Info("%s: verification of %x:%s completed successfully",
				cmd.Name(),
				snap.Header.GetIndexShortID(),
				pathname)
		}

		snap.Close()
	}

	if failures {
		return 1, fmt.Errorf("check failed")
	}

	return 0, nil
}
