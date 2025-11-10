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
	"encoding/hex"
	"flag"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/dustin/go-humanize"
)

type Rm struct {
	subcommands.SubcommandBase

	LocateOptions *locate.LocateOptions

	Apply bool
}

func init() {
	subcommands.MustRegister(func() subcommands.Subcommand { return &Rm{} }, subcommands.AgentSupport, "rm")
}

func (cmd *Rm) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.LocateOptions = locate.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("rm", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] SNAPSHOT...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.BoolVar(&cmd.Apply, "apply", false, "do the actual removal")
	cmd.LocateOptions.InstallDeletionFlags(flags)
	flags.Parse(args)

	if flags.NArg() == 0 && cmd.LocateOptions.Empty() {
		return fmt.Errorf("no filter specified, not going to remove everything")
	}

	cmd.LocateOptions.Filters.IDs = flags.Args()

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *Rm) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	matches, err := locate.LocateSnapshotIDs(repo, cmd.LocateOptions)
	if err != nil {
		return 1, err
	}

	if len(matches) == 0 {
		ctx.GetLogger().Info("rm: no snapshots matched the selection")
		return 0, nil
	}

	// plan
	if !cmd.Apply {
		type planEntry struct {
			prefix string
			id     objects.MAC
			key    string
			ts     time.Time
		}

		entries := make([]planEntry, 0, len(matches))
		for _, id := range matches {
			key := fmt.Sprintf("%x", id[:])
			snap, err := snapshot.Load(repo, id)
			if err != nil {
				ctx.GetLogger().Warn("rm: skipping %x for timestamp lookup: %v", id[:4], err)
				continue
			}

			tags := ""
			tagList := strings.Join(snap.Header.Tags, ",")
			if tagList != "" {
				tags = " tags=" + strings.Join(snap.Header.Tags, ",")
			}
			prefix := fmt.Sprintf("%s %10s%10s%10s %s%s",
				snap.Header.Timestamp.UTC().Format(time.RFC3339),
				hex.EncodeToString(snap.Header.GetIndexShortID()),
				humanize.IBytes(snap.Header.GetSource(0).Summary.Directory.Size+snap.Header.GetSource(0).Summary.Below.Size),
				snap.Header.Duration.Round(time.Second),
				utils.SanitizeText(snap.Header.GetSource(0).Importer.Directory),
				tags)
			snap.Close()
			entries = append(entries, planEntry{prefix: prefix, id: id, key: key, ts: snap.Header.Timestamp})
		}

		// Sort newest-first; unknown timestamps (IsZero) go last
		sort.SliceStable(entries, func(i, j int) bool {
			ti, tj := entries[i].ts, entries[j].ts
			if ti.IsZero() && tj.IsZero() {
				return entries[i].key < entries[j].key // stable tiebreak
			}
			if ti.IsZero() {
				return false
			}
			if tj.IsZero() {
				return true
			}
			return ti.After(tj)
		})
		fmt.Fprintf(ctx.Stdout, "rm: would remove these %d snapshot(s), run with -apply to proceed\n", len(matches))
		l := 0
		for _, e := range entries {
			l = max(l, len(e.prefix))
		}
		for _, e := range entries {
			fmt.Fprintf(ctx.Stdout, "%s\n", e.prefix)
		}
		return 0, nil
	}

	// execution
	errors := 0
	wg := sync.WaitGroup{}
	for _, matchID := range matches {
		wg.Add(1)
		go func(snapshotID objects.MAC) {
			defer wg.Done()
			if err := repo.DeleteSnapshot(snapshotID); err != nil {
				ctx.GetLogger().Error("%s", err)
				errors++
				return
			}
			ctx.GetLogger().Info("rm: removal of %x completed successfully", snapshotID[:4])
		}(matchID)
	}
	wg.Wait()

	if errors != 0 {
		return 1, fmt.Errorf("failed to remove %d snapshots", errors)
	}

	return 0, nil
}
