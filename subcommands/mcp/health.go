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

package mcp

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
)

// repositoryHealthInput mirrors no single subcommand: it condenses what info,
// locks and the snapshot list say into the one answer that matters, "are my
// backups in good shape".
type repositoryHealthInput struct{}

type snapshotRef struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp" jsonschema:"RFC3339 creation time"`
	Age       string `json:"age" jsonschema:"how long ago the snapshot was taken"`
}

type sourceHealth struct {
	Directory string      `json:"directory" jsonschema:"source directory the snapshots were taken from"`
	Snapshots int         `json:"snapshots" jsonschema:"number of snapshots of this source"`
	Latest    snapshotRef `json:"latest" jsonschema:"most recent snapshot of this source"`
}

type repositoryHealthOutput struct {
	Snapshots   int            `json:"snapshots"`
	Newest      *snapshotRef   `json:"newest,omitempty" jsonschema:"most recent snapshot in the repository"`
	Oldest      *snapshotRef   `json:"oldest,omitempty" jsonschema:"oldest snapshot in the repository"`
	LogicalSize int64          `json:"logical_size" jsonschema:"total logical size of all snapshots in bytes"`
	StorageSize int64          `json:"storage_size" jsonschema:"bytes actually occupied in the store"`
	Locks       int            `json:"locks" jsonschema:"locks currently held on the repository"`
	Sources     []sourceHealth `json:"sources" jsonschema:"per-source breakdown, to spot a source whose backups stopped"`
}

// listTagsInput mirrors the tag summaries of "plakar locate".
type listTagsInput struct{}

type tagInfo struct {
	Tag       string `json:"tag"`
	Snapshots int    `json:"snapshots" jsonschema:"number of snapshots carrying the tag"`
}

type listTagsOutput struct {
	Tags []tagInfo `json:"tags"`
}

func newSnapshotRef(id string, ts, now time.Time) snapshotRef {
	return snapshotRef{
		ID:        id,
		Timestamp: ts.UTC().Format(time.RFC3339),
		Age:       now.Sub(ts).Truncate(time.Second).String(),
	}
}

func repositoryHealth(ctx context.Context, repo *repository.Repository) (repositoryHealthOutput, error) {
	info, err := repositoryInfo(ctx, repo)
	if err != nil {
		return repositoryHealthOutput{}, err
	}

	lockIDs, err := repo.GetLocks()
	if err != nil {
		return repositoryHealthOutput{}, err
	}

	snapshotIDs, err := locate.LocateSnapshotIDs(repo, locate.NewDefaultLocateOptions())
	if err != nil {
		return repositoryHealthOutput{}, fmt.Errorf("could not fetch snapshots list: %w", err)
	}

	out := repositoryHealthOutput{
		Snapshots:   len(snapshotIDs),
		LogicalSize: info.LogicalSize,
		StorageSize: info.StorageSize,
		Locks:       len(lockIDs),
		Sources:     make([]sourceHealth, 0),
	}

	now := time.Now()
	bySource := make(map[string]*sourceHealth)
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			return repositoryHealthOutput{}, fmt.Errorf("could not load snapshot: %w", err)
		}

		id := hex.EncodeToString(snap.Header.GetIndexShortID())
		ts := snap.Header.Timestamp
		directory := snap.Header.GetSource(0).Importer.Directory
		snap.Close()

		ref := newSnapshotRef(id, ts, now)
		if out.Newest == nil || ref.Timestamp > out.Newest.Timestamp {
			out.Newest = &ref
		}
		if out.Oldest == nil || ref.Timestamp < out.Oldest.Timestamp {
			out.Oldest = &ref
		}

		source, ok := bySource[directory]
		if !ok {
			bySource[directory] = &sourceHealth{Directory: directory, Snapshots: 1, Latest: ref}
			continue
		}
		source.Snapshots++
		if ref.Timestamp > source.Latest.Timestamp {
			source.Latest = ref
		}
	}

	for _, source := range bySource {
		out.Sources = append(out.Sources, *source)
	}
	sort.Slice(out.Sources, func(i, j int) bool {
		return out.Sources[i].Directory < out.Sources[j].Directory
	})

	return out, nil
}

func listTags(repo *repository.Repository) (listTagsOutput, error) {
	snapshotIDs, err := locate.LocateSnapshotIDs(repo, locate.NewDefaultLocateOptions())
	if err != nil {
		return listTagsOutput{}, fmt.Errorf("could not fetch snapshots list: %w", err)
	}

	counts := make(map[string]int)
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			return listTagsOutput{}, fmt.Errorf("could not load snapshot: %w", err)
		}
		for _, tag := range snap.Header.Tags {
			counts[tag]++
		}
		snap.Close()
	}

	out := listTagsOutput{Tags: make([]tagInfo, 0, len(counts))}
	for tag, n := range counts {
		out.Tags = append(out.Tags, tagInfo{Tag: tag, Snapshots: n})
	}
	sort.Slice(out.Tags, func(i, j int) bool { return out.Tags[i].Tag < out.Tags[j].Tag })

	return out, nil
}
