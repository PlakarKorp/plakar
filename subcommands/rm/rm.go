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

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/policy"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/locate"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/dustin/go-humanize"
)

type Rm struct {
	subcommands.SubcommandBase

	LocateOptions *locate.LocateOptions
	PolicyOptions *policy.PolicyOptions

	Snapshots []string

	Plan bool
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Rm{} }, subcommands.AgentSupport, "rm")
}

func (cmd *Rm) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.LocateOptions = locate.NewDefaultLocateOptions()
	cmd.PolicyOptions = policy.NewDefaultPolicyOptions()

	flags := flag.NewFlagSet("rm", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] SNAPSHOT...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.BoolVar(&cmd.Plan, "plan", false, "show what would be removed (dry-run)")
	cmd.LocateOptions.InstallFlags(flags)
	cmd.PolicyOptions.InstallFlags(flags)
	flags.Parse(args)

	if flags.NArg() != 0 && !cmd.LocateOptions.Empty() {
		ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
	} else if flags.NArg() == 0 && cmd.LocateOptions.Empty() && cmd.PolicyOptions.Empty() {
		return fmt.Errorf("no filter specified, not going to remove everything")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Snapshots = flags.Args()

	return nil
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

	if len(snapshots) == 0 {
		ctx.GetLogger().Info("rm: no snapshots matched the selection")
		return 0, nil
	}

	var toDelete []objects.MAC
	var reasons map[string]policy.Reason
	var planned map[string]struct{}

	tsByID := make(map[string]time.Time, len(snapshots))

	if cmd.PolicyOptions.Empty() {
		// No policy provided → default behavior: delete everything selected.
		toDelete = append(toDelete, snapshots...)
	} else {
		// Build policy items with timestamps
		items := make([]policy.Item, 0, len(snapshots))
		for _, id := range snapshots {
			snap, err := snapshot.Load(repo, id)
			if err != nil {
				ctx.GetLogger().Warn("rm: skipping %x for policy evaluation: %v", id[:4], err)
				continue
			}
			ts := snap.Header.Timestamp
			snap.Close()

			key := fmt.Sprintf("%x", id[:])

			items = append(items, policy.Item{
				ItemID:    key, // stable string key for reasons map
				Timestamp: ts,
			})

			tsByID[key] = ts
		}

		now := time.Now().UTC()
		kept, rs := cmd.PolicyOptions.Select(items, now)
		reasons = rs

		// any item not in kept → delete
		planned = make(map[string]struct{}, len(items))
		for _, it := range items {
			if _, ok := kept[it.ItemID]; !ok {
				planned[it.ItemID] = struct{}{}
			}
		}

		// Map back to objects.MAC in original order for stable UX
		for _, id := range snapshots {
			key := fmt.Sprintf("%x", id[:])
			if _, ok := planned[key]; ok {
				toDelete = append(toDelete, id)
			}
		}
	}

	if cmd.Plan {
		type planEntry struct {
			prefix string
			id     objects.MAC
			key    string
			ts     time.Time

			reason policy.Reason
			action string // "keep" or "delete"
		}

		// Build entries
		entries := make([]planEntry, 0, len(snapshots))
		for _, id := range snapshots {
			key := fmt.Sprintf("%x", id[:])
			snap, err := snapshot.Load(repo, id)
			if err != nil {
				ctx.GetLogger().Warn("rm -plan: skipping %x for timestamp lookup: %v", id[:4], err)
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
			entry := planEntry{prefix: prefix, id: id, key: key, ts: snap.Header.Timestamp}

			if cmd.PolicyOptions.Empty() {
				entry.action = "delete"
				if cmd.LocateOptions.Empty() {
					entry.reason = policy.Reason{Action: "delete", Note: "requested explicitly"}
				} else {
					entry.reason = policy.Reason{Action: "delete", Note: "matches location filter"}
				}
			} else {
				r, ok := reasons[key]
				// Default to "skip" if we couldn't evaluate (e.g., missing timestamp)
				entry.reason = r
				entry.action = "delete"
				if _, del := planned[key]; !del {
					entry.action = "keep"
				}
				if !ok {
					entry.reason = policy.Reason{Action: "delete", Note: "not evaluated by policy"}
					entry.action = "skip"
				}
			}
			entries = append(entries, entry)
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
		fmt.Fprintf(ctx.Stdout, "rm -plan: would remove %d snapshot(s)\n", len(toDelete))
		fmt.Fprint(ctx.Stdout, "rm -plan: policy evaluation results:\n")
		l := 0
		for _, e := range entries {
			l = max(l, len(e.prefix))
		}
		for _, e := range entries {
			for len(e.prefix) < l {
				e.prefix += " "
			}
			r := e.reason
			if r.Rule == "" {
				fmt.Fprintf(ctx.Stdout, "%s action=%s  note=%s\n", e.prefix, e.action, r.Note)
			} else {
				fmt.Fprintf(ctx.Stdout, "%s action=%s  rule=%s bucket=%s rank=%d cap=%d note=%s\n",
					e.prefix, e.action, r.Rule, r.Bucket, r.Rank, r.Cap, r.Note)
			}
		}
		return 0, nil
	}

	// EXECUTION (not a plan): delete only the ones in toDelete
	if len(toDelete) == 0 {
		ctx.GetLogger().Info("rm: nothing to remove")
		return 0, nil
	}

	sort.SliceStable(toDelete, func(i, j int) bool {
		ki := fmt.Sprintf("%x", toDelete[i][:])
		kj := fmt.Sprintf("%x", toDelete[j][:])
		// best-effort timestamp fetch; if it fails, push unknowns last
		ti := time.Time{}
		tj := time.Time{}
		if snap, err := snapshot.Load(repo, toDelete[i]); err == nil {
			ti = snap.Header.Timestamp
			snap.Close()
		}
		if snap, err := snapshot.Load(repo, toDelete[j]); err == nil {
			tj = snap.Header.Timestamp
			snap.Close()
		}
		if ti.IsZero() && tj.IsZero() {
			return ki < kj
		}
		if ti.IsZero() {
			return false
		}
		if tj.IsZero() {
			return true
		}
		return ti.After(tj)
	})

	errors := 0
	wg := sync.WaitGroup{}
	for _, snap := range toDelete {
		wg.Add(1)
		go func(snapshotID objects.MAC) {
			defer wg.Done()
			if err := repo.DeleteSnapshot(snapshotID); err != nil {
				ctx.GetLogger().Error("%s", err)
				errors++
				return
			}
			ctx.GetLogger().Info("rm: removal of %x completed successfully", snapshotID[:4])
		}(snap)
	}
	wg.Wait()

	if errors != 0 {
		return 1, fmt.Errorf("failed to remove %d snapshots", errors)
	}

	return 0, nil
}
