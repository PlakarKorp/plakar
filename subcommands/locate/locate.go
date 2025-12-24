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

	plocate "github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Locate{} }, 0, "locate")
}

func (cmd *Locate) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.LocateOptions = plocate.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("locate", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] PATTERN...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&cmd.Snapshot, "snapshot", "", "snapshot to locate in")
	cmd.LocateOptions.InstallLocateFlags(flags)
	flags.Parse(args)

	if cmd.Snapshot != "" && !cmd.LocateOptions.Empty() {
		ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Patterns = flags.Args()

	return nil
}

type Locate struct {
	subcommands.SubcommandBase

	LocateOptions *plocate.LocateOptions
	Snapshot      string
	Patterns      []string
}

func (cmd *Locate) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var snapshots []objects.MAC
	if len(cmd.Snapshot) == 0 {
		snapshotIDs, err := plocate.LocateSnapshotIDs(repo, cmd.LocateOptions)
		if err != nil {
			return 1, fmt.Errorf("ls: could not fetch snapshots list: %w", err)
		}
		snapshots = append(snapshots, snapshotIDs...)
	} else {
		snapshotIDs := plocate.LookupSnapshotByPrefix(repo, cmd.Snapshot)
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

			if err := ctx.Err(); err != nil {
				return 1, err
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
				fmt.Printf("%x:%s\n", snap.Header.Identifier[0:4], utils.SanitizeText(pathname))
			}
		}
		snap.Close()
	}
	return 0, nil
}
