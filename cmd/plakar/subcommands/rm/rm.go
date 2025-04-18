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

package rm

import (
	"flag"
	"fmt"
	"sync"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
)

func init() {
	subcommands.Register("rm", parse_cmd_rm)
}

func parse_cmd_rm(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	locateopts := utils.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("rm", flag.ExitOnError)
	locateopts.InstallFlags(flags)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] SNAPSHOT...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.Parse(args)

	if flags.NArg() != 0 && !locateopts.Empty() {
		ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
	} else if flags.NArg() == 0 && locateopts.Empty() {
		return nil, fmt.Errorf("no filter specified, not going to remove everything")
	}

	return &Rm{
		RepositorySecret: ctx.GetSecret(),

		LocateOptions: locateopts,
		Snapshots: flags.Args(),
	}, nil
}

type Rm struct {
	RepositorySecret []byte

	LocateOptions *utils.LocateOptions
	Snapshots     []string
}

func (cmd *Rm) Name() string {
	return "rm"
}

func (cmd *Rm) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var snapshots []objects.MAC
	if len(cmd.Snapshots) == 0 {
		snapshotIDs, err := utils.LocateSnapshotIDs(repo, cmd.LocateOptions)
		if err != nil {
			return 1, err
		}
		snapshots = append(snapshots, snapshotIDs...)
	} else {
		for _, prefix := range cmd.Snapshots {
			snapshotID, err := utils.LocateSnapshotByPrefix(repo, prefix)
			if err != nil {
				continue
			}
			snapshots = append(snapshots, snapshotID)
		}
	}

	errors := 0
	wg := sync.WaitGroup{}
	for _, snap := range snapshots {
		wg.Add(1)
		go func(snapshotID objects.MAC) {
			err := repo.DeleteSnapshot(snapshotID)
			if err != nil {
				ctx.GetLogger().Error("%s", err)
				errors++
			}
			ctx.GetLogger().Info("%s: removal of %x completed successfully",
				cmd.Name(),
				snapshotID[:4])
			wg.Done()
		}(snap)
	}
	wg.Wait()

	if errors != 0 {
		return 1, fmt.Errorf("failed to remove %d snapshots", errors)
	}

	return 0, nil
}
