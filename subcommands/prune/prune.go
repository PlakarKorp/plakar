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

package prune

import (
	"encoding/hex"
	"flag"
	"fmt"
	"path/filepath"
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

func init() {
	subcommands.Register(Prune, 0, "prune")
}

// mergePolicyOptions layers CLI overrides (from) onto policy-loaded options
// (to): for each field, the CLI value wins iff it is set, otherwise the
// policy value is kept.
func mergePolicyOptions(to *locate.LocateOptions, from *locate.LocateOptions) {
	mergeFilters(&to.Filters, &from.Filters)

	if from.GroupBy != locate.GroupByNone {
		to.GroupBy = from.GroupBy
	}

	merge := func(a, b *locate.LocatePeriod) {
		if b.Keep != 0 {
			a.Keep = b.Keep
		}
		if b.Cap != 0 {
			a.Cap = b.Cap
		}
	}

	merge(&to.Periods.Minute, &from.Periods.Minute)
	merge(&to.Periods.Hour, &from.Periods.Hour)
	merge(&to.Periods.Day, &from.Periods.Day)
	merge(&to.Periods.Week, &from.Periods.Week)
	merge(&to.Periods.Month, &from.Periods.Month)
	merge(&to.Periods.Year, &from.Periods.Year)
	merge(&to.Periods.Monday, &from.Periods.Monday)
	merge(&to.Periods.Tuesday, &from.Periods.Tuesday)
	merge(&to.Periods.Wednesday, &from.Periods.Wednesday)
	merge(&to.Periods.Thursday, &from.Periods.Thursday)
	merge(&to.Periods.Friday, &from.Periods.Friday)
	merge(&to.Periods.Saturday, &from.Periods.Saturday)
	merge(&to.Periods.Sunday, &from.Periods.Sunday)

}

// mergeFilters overlays b onto a per-field: each field of a is replaced only
// when the corresponding field in b is non-zero. Slice fields are replaced
// wholesale when b's slice is non-empty.
func mergeFilters(a, b *locate.LocateFilters) {
	if !b.Before.IsZero() {
		a.Before = b.Before
	}
	if !b.Since.IsZero() {
		a.Since = b.Since
	}
	if b.Name != "" {
		a.Name = b.Name
	}
	if b.Category != "" {
		a.Category = b.Category
	}
	if b.Environment != "" {
		a.Environment = b.Environment
	}
	if b.Perimeter != "" {
		a.Perimeter = b.Perimeter
	}
	if b.Job != "" {
		a.Job = b.Job
	}
	if len(b.Tags) > 0 {
		a.Tags = b.Tags
	}
	if len(b.IgnoreTags) > 0 {
		a.IgnoreTags = b.IgnoreTags
	}
	if b.Latest {
		a.Latest = b.Latest
	}
	if len(b.IDs) > 0 {
		a.IDs = b.IDs
	}
	if len(b.Types) > 0 {
		a.Types = b.Types
	}
	if len(b.Origins) > 0 {
		a.Origins = b.Origins
	}
	if len(b.Roots) > 0 {
		a.Roots = b.Roots
	}
}

type planEntry struct {
	prefix string
	id     objects.MAC
	key    string
	ts     time.Time

	reason locate.Reason
	action string // "keep" or "delete"
}

func Prune(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		locopts        = locate.NewDefaultLocateOptions()
		policyOverride = locate.NewDefaultLocateOptions()
		apply          bool
		policyName     string
	)

	flags := flag.NewFlagSet("prune", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] SNAPSHOT...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.BoolVar(&apply, "apply", false, "do the actual removal")
	flags.StringVar(&policyName, "policy", "", "policy to use")
	policyOverride.InstallLocateFlags(flags)
	flags.Parse(args)

	if policyName != "" {
		configFile := filepath.Join(ctx.ConfigDir, "policies.yml")
		cfg, err := utils.LoadPolicyConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to load policies config: %w", err)
		}
		if !cfg.Has(policyName) {
			return fmt.Errorf("policy %q not found", policyName)
		}
		cfg.ApplyConfig(policyName, locopts)
	}
	mergePolicyOptions(locopts, policyOverride)

	if flags.NArg() == 0 && locopts.Empty() {
		return fmt.Errorf("no filter specified, not going to prune everything")
	}

	_, reasons, err := locate.Match(repo, locopts)
	if err != nil {
		return err
	}

	toDelete := make([]objects.MAC, 0, len(reasons))
	entries := make([]planEntry, 0, len(reasons))

	for id, r := range reasons {
		if r.Action == "delete" {
			toDelete = append(toDelete, id)
		}

		snap, err := snapshot.Load(repo, id)
		if err != nil {
			ctx.GetLogger().Warn("prune: skipping %x for timestamp lookup: %v", id[:4], err)
			continue
		}

		tags := ""
		tagList := strings.Join(snap.Header.Tags, ",")
		if tagList != "" {
			tags = " tags=" + strings.Join(snap.Header.Tags, ",")
		}
		prefix := fmt.Sprintf("%s %10s%10s %s%s",
			snap.Header.Timestamp.UTC().Format(time.RFC3339),
			hex.EncodeToString(snap.Header.GetIndexShortID()),
			humanize.IBytes(snap.Header.GetSource(0).Summary.Directory.Size+snap.Header.GetSource(0).Summary.Below.Size),
			utils.SanitizeText(snap.Header.GetSource(0).Importer.Directory),
			tags)
		snap.Close()
		entry := planEntry{prefix: prefix, id: id, key: r.Bucket, ts: snap.Header.Timestamp}

		r, ok := reasons[id]
		// Default to "skip" if we couldn't evaluate (e.g., missing timestamp)
		entry.reason = r
		entry.action = r.Action
		if !ok {
			entry.reason = locate.Reason{Action: "delete", Note: "not evaluated by policy"}
			entry.action = "skip"
		}
		entries = append(entries, entry)
	}

	if !apply {
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
		fmt.Fprintf(ctx.Stdout, "prune: would keep %d and delete %d snapshot(s), run with -apply to proceed\n", len(reasons)-len(toDelete), len(toDelete))
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
				fmt.Fprintf(ctx.Stdout, "%-8s %s  reason=%s\n", e.action, e.prefix, e.reason.Note)
			} else {
				fmt.Fprintf(ctx.Stdout, "%-8s %s  match=%s:%s rank=%d cap=%d\n",
					e.action, e.prefix, r.Rule, r.Bucket, r.Rank, r.Cap)
			}
		}
		return nil
	}

	if len(toDelete) == 0 {
		return nil
	}

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
			ctx.GetLogger().Info("prune: removal of %x completed successfully", snapshotID[:4])
		}(snap)
	}
	wg.Wait()

	if errors != 0 {
		return fmt.Errorf("failed to remove %d snapshots", errors)
	}

	return nil
}
