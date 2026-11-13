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

// Package journal records what plakar operations do.
//
// State and its fold are a copy of plaklet's (state.go and listener.go in
// PlakarKorp/plaklet), without the process samplers plaklet adds on top.
// The copy is deliberate: the two will be unified once plakar's needs are
// known, see the master plan for the operations journal.
package journal

import (
	"maps"
	"slices"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/objects"
)

// State is what the kloset event stream of one run folds into: counters per
// kind of item, the current phase, the latest per-scope I/O stats and the
// final result.
type State struct {
	SnapshotID string `json:"snapshot_id,omitzero"`
	Phase      string `json:"phase,omitzero"`

	Summary struct {
		Exists      bool   `json:"exists,omitzero"`
		Paths       uint64 `json:"paths,omitzero"`
		Files       uint64 `json:"files,omitzero"`
		Directories uint64 `json:"directories,omitzero"`
		Symlinks    uint64 `json:"symlinks,omitzero"`
		Xattrs      uint64 `json:"xattrs,omitzero"`
		Size        uint64 `json:"size,omitzero"`
	} `json:"summary,omitzero"`

	Paths    Counter `json:"paths,omitzero"`
	Dirs     Counter `json:"dirs,omitzero"`
	Files    Counter `json:"files,omitzero"`
	Xattrs   Counter `json:"xattrs,omitzero"`
	Symlinks Counter `json:"symlinks,omitzero"`
	Chunks   Counter `json:"chunks,omitzero"`
	Objects  Counter `json:"objects,omitzero"`

	Result struct {
		Size     uint64  `json:"size,omitzero"`
		Errors   uint64  `json:"errors,omitzero"`
		Duration float64 `json:"duration,omitzero"`
	} `json:"result,omitzero"`

	// Bytes/items processed so far. The total is unknown up front, so this is a
	// running count for a progress readout, not a percentage.
	Processed struct {
		Items uint64 `json:"items,omitzero"`
		Bytes uint64 `json:"bytes,omitzero"`
	} `json:"processed,omitzero"`

	// Latest per-scope I/O stats, keyed by kloset iostat scope name, from the
	// "iostats" events.
	IO map[string]IOScope `json:"io,omitzero"`

	RecentPaths []RecentPath `json:"recent_paths,omitzero"`
}

type Counter struct {
	Total      uint64 `json:"total,omitzero"`
	Ok         uint64 `json:"ok,omitzero"`
	Error      uint64 `json:"error,omitzero"`
	Size       uint64 `json:"size,omitzero"`
	Cached     uint64 `json:"cached,omitzero"`
	CachedSize uint64 `json:"cached_size,omitzero"`
}

// IODir is one direction (read or write) of a kloset iostat scope. Overall is
// active-time throughput; OverallWall is wall-clock throughput (bytes/sec).
type IODir struct {
	TotalBytes  int64   `json:"total,omitzero"`
	Overall     float64 `json:"overall,omitzero"`
	OverallWall float64 `json:"overall_wall,omitzero"`

	Latency Latency `json:"latency,omitzero"`
}

// Latency is the distribution of individual operation durations in one
// direction of a scope. kloset observes it on packfile writes to the storage
// backend only, so other scopes stay zero.
type Latency struct {
	Count int64         `json:"count,omitzero"`
	Avg   time.Duration `json:"avg,omitzero"`
	P50   time.Duration `json:"p50,omitzero"`
	P95   time.Duration `json:"p95,omitzero"`
	Max   time.Duration `json:"max,omitzero"`
}

// IOScope holds the latest read/write stats of one kloset iostat scope.
type IOScope struct {
	Read  IODir `json:"r,omitzero"`
	Write IODir `json:"w,omitzero"`
}

// maxRecentPaths caps RecentPaths: in-progress paths first, then the most
// recently settled ones.
const maxRecentPaths = 5

type RecentPath struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

// processed fills the running items/bytes readout from the per-file counters,
// falling back to the scan summary (check and export emit fs.summary; backup
// does not).
func (s *State) processed() (items, bytes uint64) {
	items = s.Files.Ok + s.Files.Cached
	bytes = s.Files.Size + s.Files.CachedSize
	if bytes == 0 {
		bytes = s.Summary.Size
	}
	if items == 0 {
		items = s.Summary.Files
	}
	return
}

// Clone returns a copy the caller owns, with the running Processed counters
// filled in.
func (s *State) Clone() State {
	c := *s
	c.Processed.Items, c.Processed.Bytes = s.processed()
	c.IO = maps.Clone(s.IO)
	c.RecentPaths = slices.Clone(s.RecentPaths)
	return c
}

// Update folds one event into the state.
func (s *State) Update(e *events.Event) {
	switch e.Type {
	case "workflow.start":
		if e.Snapshot != objects.NilMac {
			s.SnapshotID = e.Snapshot.FormatHex()
		}
	case "workflow.end":

	case "path":
		s.Paths.Total++
		pathName, _ := eventField[string](e, "path")
		s.startRecentPath(pathName)
	case "path.ok":
		s.Paths.Ok++
		pathName, _ := eventField[string](e, "path")
		s.settleRecentPath(pathName, "ok")
	case "path.error":
		s.Paths.Error++
		pathName, _ := eventField[string](e, "path")
		s.settleRecentPath(pathName, "error")
	case "path.cached":
		s.Paths.Cached++
		pathName, _ := eventField[string](e, "path")
		s.settleRecentPath(pathName, "ok")

	case "directory":
		s.Dirs.Total++
	case "directory.ok":
		s.Dirs.Ok++
	case "directory.error":
		s.Dirs.Error++
	case "directory.cached":
		s.Dirs.Cached++

	case "file":
		s.Files.Total++
	case "file.ok":
		s.Files.Ok++
		fi, _ := eventField[objects.FileInfo](e, "fileinfo")
		s.Files.Size += uint64(fi.Size())
	case "file.error":
		s.Files.Error++
	case "file.cached":
		s.Files.Cached++
		fi, _ := eventField[objects.FileInfo](e, "fileinfo")
		s.Files.CachedSize += uint64(fi.Size())

	case "xattr":
		s.Xattrs.Total++
	case "xattr.ok":
		s.Xattrs.Ok++
		size, _ := eventField[int64](e, "size")
		s.Xattrs.Size += uint64(size)
	case "xattr.error":
		s.Xattrs.Error++
	case "xattr.cached":
		s.Xattrs.Cached++
		size, _ := eventField[int64](e, "size")
		s.Xattrs.CachedSize += uint64(size)

	case "symlink":
		s.Symlinks.Total++
	case "symlink.ok":
		s.Symlinks.Ok++
	case "symlink.error":
		s.Symlinks.Error++
	case "symlink.cached":
		s.Symlinks.Cached++

	case "object":
		s.Objects.Total++
	case "object.ok":
		s.Objects.Ok++
	case "object.error":
		s.Objects.Error++
	case "object.cached":
		s.Objects.Cached++

	case "chunk":
		s.Chunks.Total++
	case "chunk.ok":
		s.Chunks.Ok++
	case "chunk.error":
		s.Chunks.Error++
	case "chunk.cached":
		s.Chunks.Cached++

	case "fs.summary":
		s.Summary.Exists = true
		s.Summary.Files, _ = eventField[uint64](e, "files")
		s.Summary.Directories, _ = eventField[uint64](e, "directories")
		s.Summary.Symlinks, _ = eventField[uint64](e, "symlinks")
		s.Summary.Xattrs, _ = eventField[uint64](e, "xattrs")
		s.Summary.Size, _ = eventField[uint64](e, "size")
		s.Summary.Paths = s.Summary.Files + s.Summary.Directories + s.Summary.Symlinks + s.Summary.Xattrs

	case "snapshot.import.start":
		s.Phase = "processing"
	case "snapshot.import.done":
	case "snapshot.vfs.start":
		s.Phase = "building VFS"
	case "snapshot.vfs.end":
		s.Phase = ""
	case "snapshot.index.start":
		s.Phase = "indexing"
	case "snapshot.index.end":
		s.Phase = ""
	case "snapshot.commit.start":
		s.Phase = "committing"

	case "iostats":
		scope, ok := eventField[string](e, "scope")
		if !ok || scope == "" {
			return
		}
		if s.IO == nil {
			s.IO = make(map[string]IOScope)
		}
		s.IO[scope] = IOScope{
			Read:  ioDirFromEvent(e, "r"),
			Write: ioDirFromEvent(e, "w"),
		}

	case "result":
		s.Phase = "completed"
		s.Result.Size, _ = eventField[uint64](e, "size")
		s.Result.Errors, _ = eventField[uint64](e, "errors")
		if d, ok := eventField[time.Duration](e, "duration"); ok {
			s.Result.Duration = d.Seconds()
		}
	}
}

// ioDirFromEvent pulls one direction ("r"/"w") of an iostats event into an
// IODir; missing or mistyped fields default to zero.
func ioDirFromEvent(e *events.Event, key string) IODir {
	dir, _ := eventField[map[string]any](e, key)
	getf := func(k string) float64 {
		switch n := dir[k].(type) {
		case float64:
			return n
		case int64:
			return float64(n)
		default:
			return 0
		}
	}
	return IODir{
		TotalBytes:  int64(getf("total")),
		Overall:     getf("overall"),
		OverallWall: getf("overall_wall"),
		Latency:     latencyFromDir(dir),
	}
}

// latencyFromDir pulls the nested "latency" map of one iostats direction. The
// bus carries Go values: durations as time.Duration, the count as int64.
func latencyFromDir(dir map[string]any) Latency {
	lat, _ := dir["latency"].(map[string]any)
	if lat == nil {
		return Latency{}
	}
	getd := func(k string) time.Duration {
		d, _ := lat[k].(time.Duration)
		return d
	}
	count, _ := lat["count"].(int64)
	return Latency{
		Count: count,
		Avg:   getd("avg"),
		P50:   getd("p50"),
		P95:   getd("p95"),
		Max:   getd("max"),
	}
}

// startRecentPath puts a path that started processing in front of RecentPaths.
func (s *State) startRecentPath(path string) {
	inProgress := []RecentPath{{Path: path, Status: "in-progress"}}
	var settled []RecentPath
	for _, rp := range s.RecentPaths {
		if rp.Status == "in-progress" {
			inProgress = append(inProgress, rp)
		} else {
			settled = append(settled, rp)
		}
	}
	s.RecentPaths = capRecentPaths(append(inProgress, settled...))
}

// settleRecentPath moves a path from in-progress to the head of the settled
// ones, with its final status.
func (s *State) settleRecentPath(path string, status string) {
	var inProgress []RecentPath
	settled := []RecentPath{{Path: path, Status: status}}
	for _, rp := range s.RecentPaths {
		if rp.Path == path {
			continue
		}
		if rp.Status == "in-progress" {
			inProgress = append(inProgress, rp)
		} else {
			settled = append(settled, rp)
		}
	}
	s.RecentPaths = capRecentPaths(append(inProgress, settled...))
}

func capRecentPaths(paths []RecentPath) []RecentPath {
	if len(paths) > maxRecentPaths {
		return paths[:maxRecentPaths]
	}
	return paths
}

// eventField extracts a typed value from an event's Data map. It returns the
// zero value and false when the key is missing or holds a different type, so
// a malformed field is skipped instead of panicking on a bad assertion.
func eventField[T any](e *events.Event, key string) (T, bool) {
	v, ok := e.Data[key].(T)
	return v, ok
}
