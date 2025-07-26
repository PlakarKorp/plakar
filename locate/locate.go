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
	"encoding/hex"
	"flag"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/utils"
)

type locateSortOrder int

const (
	LocateSortOrderNone       locateSortOrder = 0
	LocateSortOrderAscending  locateSortOrder = 1
	LocateSortOrderDescending locateSortOrder = -1
)

type LocateOptions struct {
	MaxConcurrency int
	SortOrder      locateSortOrder
	Latest         bool

	Before time.Time
	Since  time.Time

	Name        string
	Category    string
	Environment string
	Perimeter   string
	Job         string
	Tag         string

	Prefix string
}

func NewDefaultLocateOptions() *LocateOptions {
	return &LocateOptions{
		MaxConcurrency: 1,
		SortOrder:      LocateSortOrderNone,
		Latest:         false,

		Before: time.Time{},
		Since:  time.Time{},

		Name:        "",
		Category:    "",
		Environment: "",
		Perimeter:   "",
		Job:         "",
		Tag:         "",

		Prefix: "",
	}
}

func (lo *LocateOptions) Empty() bool {
	return *lo == *NewDefaultLocateOptions()
}

func (lo *LocateOptions) InstallFlags(flags *flag.FlagSet) {
	flags.StringVar(&lo.Name, "name", "", "filter by name")
	flags.StringVar(&lo.Category, "category", "", "filter by category")
	flags.StringVar(&lo.Environment, "environment", "", "filter by environment")
	flags.StringVar(&lo.Perimeter, "perimeter", "", "filter by perimeter")
	flags.StringVar(&lo.Job, "job", "", "filter by job")
	flags.StringVar(&lo.Tag, "tag", "", "filter by tag")

	flags.BoolVar(&lo.Latest, "latest", false, "use latest snapshot")

	flags.Var(utils.NewTimeFlag(&lo.Before), "before", "filter by date")
	flags.Var(utils.NewTimeFlag(&lo.Since), "since", "filter by date")
}

func LocateSnapshotIDs(repo *repository.Repository, opts *LocateOptions) ([]objects.MAC, error) {
	type result struct {
		snapshotID objects.MAC
		timestamp  time.Time
	}

	workSet := make([]result, 0)
	workSetMutex := sync.Mutex{}

	if opts == nil {
		opts = NewDefaultLocateOptions()
	}

	wg := sync.WaitGroup{}
	maxConcurrency := make(chan struct{}, opts.MaxConcurrency)
	for snapshotID := range repo.ListSnapshots() {
		maxConcurrency <- struct{}{}
		wg.Add(1)
		go func(snapshotID objects.MAC) {
			defer func() {
				<-maxConcurrency
				wg.Done()
			}()

			snap, err := snapshot.Load(repo, snapshotID)
			if err != nil {
				return
			}
			defer snap.Close()

			if opts.Prefix != "" {
				if !strings.HasPrefix(hex.EncodeToString(snapshotID[:]), opts.Prefix) {
					return
				}
			}

			if opts.Name != "" {
				if snap.Header.Name != opts.Name {
					return
				}
			}

			if opts.Category != "" {
				if snap.Header.Category != opts.Category {
					return
				}
			}

			if opts.Environment != "" {
				if snap.Header.Environment != opts.Environment {
					return
				}
			}

			if opts.Perimeter != "" {
				if snap.Header.Perimeter != opts.Perimeter {
					return
				}
			}

			if opts.Job != "" {
				if snap.Header.Job != opts.Job {
					return
				}
			}

			if opts.Tag != "" {
				if !snap.Header.HasTag(opts.Tag) {
					return
				}
			}

			if !opts.Before.IsZero() {
				if snap.Header.Timestamp.After(opts.Before) {
					return
				}
			}

			if !opts.Since.IsZero() {
				if snap.Header.Timestamp.Before(opts.Since) {
					return
				}
			}

			workSetMutex.Lock()
			workSet = append(workSet, result{
				snapshotID: snapshotID,
				timestamp:  snap.Header.Timestamp,
			})
			workSetMutex.Unlock()
		}(snapshotID)
	}
	wg.Wait()

	if opts.SortOrder != LocateSortOrderNone {
		if opts.SortOrder == LocateSortOrderAscending {
			sort.SliceStable(workSet, func(i, j int) bool {
				return workSet[i].timestamp.Before(workSet[j].timestamp)
			})
		} else {
			sort.SliceStable(workSet, func(i, j int) bool {
				return workSet[i].timestamp.After(workSet[j].timestamp)
			})
		}
	}

	if opts.Latest {
		if len(workSet) >= 1 {
			workSet = workSet[:1]
		}
	}

	resultSet := make([]objects.MAC, 0, len(workSet))
	for _, result := range workSet {
		resultSet = append(resultSet, result.snapshotID)
	}

	return resultSet, nil
}

func ParseSnapshotPath(snapshotPath string) (string, string) {
	if strings.HasPrefix(snapshotPath, "/") {
		return "", snapshotPath
	}
	tmp := strings.SplitN(snapshotPath, ":", 2)
	prefix := snapshotPath
	pattern := ""
	if len(tmp) == 2 {
		prefix, pattern = tmp[0], tmp[1]
	}
	return prefix, pattern
}

func LookupSnapshotByPrefix(repo *repository.Repository, prefix string) []objects.MAC {
	ret := make([]objects.MAC, 0)
	for snapshotID := range repo.ListSnapshots() {
		if strings.HasPrefix(hex.EncodeToString(snapshotID[:]), prefix) {
			ret = append(ret, snapshotID)
		}
	}
	return ret
}

func LocateSnapshotByPrefix(repo *repository.Repository, prefix string) (objects.MAC, error) {
	snapshots := LookupSnapshotByPrefix(repo, prefix)
	if len(snapshots) == 0 {
		return objects.MAC{}, fmt.Errorf("no snapshot has prefix: %s", prefix)
	}
	if len(snapshots) > 1 {
		return objects.MAC{}, fmt.Errorf("snapshot ID is ambiguous: %s (matches %d snapshots)", prefix, len(snapshots))
	}
	return snapshots[0], nil
}

func OpenSnapshotByPath(repo *repository.Repository, snapshotPath string) (*snapshot.Snapshot, string, error) {
	prefix, pathname := ParseSnapshotPath(snapshotPath)

	snapshotID, err := LocateSnapshotByPrefix(repo, prefix)
	if err != nil {
		return nil, "", err
	}

	snap, err := snapshot.Load(repo, snapshotID)
	if err != nil {
		return nil, "", err
	}

	var snapRoot string
	if strings.HasPrefix(pathname, "/") {
		snapRoot = pathname
	} else {
		snapRoot = path.Clean(path.Join(snap.Header.GetSource(0).Importer.Directory, pathname))
	}
	return snap, path.Clean(snapRoot), err
}
