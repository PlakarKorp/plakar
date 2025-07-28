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

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/locate"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Rm{} }, subcommands.AgentSupport, "rm")
}

func (cmd *Rm) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.LocateOptions = locate.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("rm", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] SNAPSHOT...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	cmd.LocateOptions.InstallFlags(flags)
	flags.Parse(args)

	if flags.NArg() != 0 && !cmd.LocateOptions.Empty() {
		ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
	} else if flags.NArg() == 0 && cmd.LocateOptions.Empty() {
		return fmt.Errorf("no filter specified, not going to remove everything")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Snapshots = flags.Args()

	return nil
}

type Rm struct {
	subcommands.SubcommandBase

	LocateOptions *locate.LocateOptions
	Snapshots     []string
}

func (cmd *Rm) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var snapshots []objects.MAC
	if len(cmd.Snapshots) == 0 {
		snapshotIDs, err := locate.LocateSnapshotIDs(repo, cmd.LocateOptions)
		if err != nil {
			return 1, err
		}
		snapshots = append(snapshots, snapshotIDs...)
	} else {
		for _, prefix := range cmd.Snapshots {
			snapshotID, err := locate.LocateSnapshotByPrefix(repo, prefix)
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
			ctx.GetLogger().Info("rm: removal of %x completed successfully", snapshotID[:4])
			wg.Done()
		}(snap)
	}
	wg.Wait()

	if errors != 0 {
		return 1, fmt.Errorf("failed to remove %d snapshots", errors)
	}

	return 0, nil
}
