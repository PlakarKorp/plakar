/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
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

package locate

import (
	"flag"
	"fmt"
	"path"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
)

func init() {
	subcommands.Register("locate", parse_cmd_locate)
}

func parse_cmd_locate(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	var opt_snapshot string
	var opt_name string
	var opt_category string
	var opt_environment string
	var opt_perimeter string
	var opt_job string
	var opt_tag string
	var opt_before string
	var opt_since string
	var opt_latest bool

	flags := flag.NewFlagSet("locate", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] PATTERN...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&opt_name, "name", "", "filter by name")
	flags.StringVar(&opt_category, "category", "", "filter by category")
	flags.StringVar(&opt_environment, "environment", "", "filter by environment")
	flags.StringVar(&opt_perimeter, "perimeter", "", "filter by perimeter")
	flags.StringVar(&opt_job, "job", "", "filter by job")
	flags.StringVar(&opt_tag, "tag", "", "filter by tag")
	flags.StringVar(&opt_before, "before", "", "filter by date")
	flags.StringVar(&opt_since, "since", "", "filter by date")
	flags.BoolVar(&opt_latest, "latest", false, "use latest snapshot")
	flags.StringVar(&opt_snapshot, "snapshot", "", "snapshot to locate in")
	flags.Parse(args)

	var err error

	var beforeDate time.Time
	if opt_before != "" {
		beforeDate, err = utils.ParseTimeFlag(opt_before)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %s", opt_before)
		}
	}

	var sinceDate time.Time
	if opt_since != "" {
		sinceDate, err = utils.ParseTimeFlag(opt_since)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %s", opt_since)
		}
	}

	if opt_snapshot != "" {
		if opt_name != "" || opt_category != "" || opt_environment != "" || opt_perimeter != "" || opt_job != "" || opt_tag != "" || !beforeDate.IsZero() || !sinceDate.IsZero() || opt_latest {
			ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
		}
	}

	return &Locate{
		RepositorySecret: ctx.GetSecret(),

		OptBefore: beforeDate,
		OptSince:  sinceDate,
		OptLatest: opt_latest,

		OptName:        opt_name,
		OptCategory:    opt_category,
		OptEnvironment: opt_environment,
		OptPerimeter:   opt_perimeter,
		OptJob:         opt_job,
		OptTag:         opt_tag,

		Snapshot: opt_snapshot,
		Patterns: flags.Args(),
	}, nil
}

type Locate struct {
	RepositorySecret []byte

	OptBefore time.Time
	OptSince  time.Time
	OptLatest bool

	OptName        string
	OptCategory    string
	OptEnvironment string
	OptPerimeter   string
	OptJob         string
	OptTag         string

	Snapshot string
	Patterns []string
}

func (cmd *Locate) Name() string {
	return "locate"
}

func (cmd *Locate) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var snapshots []objects.MAC
	if len(cmd.Snapshot) == 0 {
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
			return 1, fmt.Errorf("ls: could not fetch snapshots list: %w", err)
		}
		snapshots = append(snapshots, snapshotIDs...)
	} else {
		snapshotIDs := utils.LookupSnapshotByPrefix(repo, cmd.Snapshot)
		snapshots = append(snapshots, snapshotIDs...)
	}

	for _, snapshotID := range snapshots {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			return 1, fmt.Errorf("locate: could not get snapshot: %w", err)
		}

		fs, err := snap.Filesystem()
		if err != nil {
			snap.Close()
			return 1, fmt.Errorf("locate: could not get filesystem: %w", err)
		}
		for pathname, err := range fs.Pathnames() {
			if err != nil {
				snap.Close()
				return 1, fmt.Errorf("locate: could not get pathname: %w", err)
			}

			for _, pattern := range cmd.Patterns {
				matched := false
				if path.Base(pathname) == pattern {
					matched = true
				}
				if !matched {
					matched, err := path.Match(pattern, path.Base(pathname))
					if err != nil {
						snap.Close()
						return 1, fmt.Errorf("locate: could not match pattern: %w", err)
					}
					if !matched {
						continue
					}
				}
				fmt.Fprintf(ctx.Stdout, "%x:%s\n", snap.Header.Identifier[0:4], utils.SanitizeText(pathname))
			}
		}
		snap.Close()
	}
	return 0, nil
}
