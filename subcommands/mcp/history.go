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
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
)

// compareSnapshotsInput mirrors "plakar diff" between two whole snapshots, but
// reports structure rather than content: which entries appeared, disappeared or
// changed. diff_snapshots then shows what changed inside one file.
type compareSnapshotsInput struct {
	Snapshot1 string `json:"snapshot1" jsonschema:"older snapshot ID"`
	Snapshot2 string `json:"snapshot2" jsonschema:"newer snapshot ID"`
	Path      string `json:"path,omitempty" jsonschema:"restrict the comparison to this subtree, defaults to the snapshot root"`
}

type treeChange struct {
	Path   string `json:"path"`
	Change string `json:"change" jsonschema:"one of added, removed or modified"`
	Size   int64  `json:"size" jsonschema:"size of the entry, in snapshot2 for added and modified, in snapshot1 for removed"`
}

type compareSnapshotsOutput struct {
	Added     int          `json:"added"`
	Removed   int          `json:"removed"`
	Modified  int          `json:"modified"`
	Changes   []treeChange `json:"changes"`
	Truncated bool         `json:"truncated" jsonschema:"true when the list of changes was cut short; the counts remain exact"`
}

// fileHistoryInput answers when a file changed: every snapshot that contains
// the path, newest first, with content changes flagged.
type fileHistoryInput struct {
	Path string `json:"path" jsonschema:"path of the file, the same in every snapshot"`
}

type fileVersion struct {
	Snapshot     string `json:"snapshot" jsonschema:"short ID of the snapshot holding this version"`
	SnapshotTime string `json:"snapshot_time" jsonschema:"RFC3339 creation time of the snapshot"`
	Size         int64  `json:"size"`
	ModTime      string `json:"mod_time" jsonschema:"RFC3339 modification time recorded for the file"`
	Object       string `json:"object" jsonschema:"content identifier; two versions with the same object have identical contents"`
	Changed      bool   `json:"changed" jsonschema:"true when the content differs from the previous, older version"`
}

type fileHistoryOutput struct {
	Path      string        `json:"path"`
	Versions  []fileVersion `json:"versions" jsonschema:"every snapshot containing the file, newest first"`
	Truncated bool          `json:"truncated"`
}

// entrySig is what makes an entry "the same" across two snapshots: for a
// regular file the content-addressed object, for a symlink its target.
type entrySig struct {
	object  string
	symlink string
	mode    fs.FileMode
	size    int64
}

// maxCompareEntries bounds the per-side index compare_snapshots builds in
// memory; a snapshot bigger than this must be compared subtree by subtree.
const maxCompareEntries = 250000

func collectTree(repo *repository.Repository, snapshotID, pathname string) (map[string]entrySig, error) {
	snap, resolved, err := locate.OpenSnapshotByPath(repo, snapshotPath(snapshotID, pathname))
	if err != nil {
		return nil, err
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		return nil, err
	}

	tree := make(map[string]entrySig)
	err = pvfs.WalkDir(resolved, func(entryPath string, d *vfs.Entry, err error) error {
		if err != nil {
			return err
		}
		// Directories carry no content of their own: comparing them would only
		// echo the changes of what they contain.
		if d.IsDir() {
			return nil
		}
		if len(tree) == maxCompareEntries {
			return fmt.Errorf("more than %d entries, restrict the comparison with path", maxCompareEntries)
		}

		tree[entryPath] = entrySig{
			object:  hex.EncodeToString(d.Object[:]),
			symlink: d.SymlinkTarget,
			mode:    d.Stat().Mode(),
			size:    d.Stat().Size(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return tree, nil
}

func compareSnapshots(repo *repository.Repository, in compareSnapshotsInput) (compareSnapshotsOutput, error) {
	if in.Snapshot1 == "" || in.Snapshot2 == "" {
		return compareSnapshotsOutput{}, fmt.Errorf("snapshot1 and snapshot2 are required")
	}

	tree1, err := collectTree(repo, in.Snapshot1, in.Path)
	if err != nil {
		return compareSnapshotsOutput{}, fmt.Errorf("snapshot1: %w", err)
	}
	tree2, err := collectTree(repo, in.Snapshot2, in.Path)
	if err != nil {
		return compareSnapshotsOutput{}, fmt.Errorf("snapshot2: %w", err)
	}

	changes := make([]treeChange, 0)
	for entryPath, sig2 := range tree2 {
		sig1, ok := tree1[entryPath]
		if !ok {
			changes = append(changes, treeChange{Path: entryPath, Change: "added", Size: sig2.size})
			continue
		}
		delete(tree1, entryPath)
		if sig1 != sig2 {
			changes = append(changes, treeChange{Path: entryPath, Change: "modified", Size: sig2.size})
		}
	}
	for entryPath, sig1 := range tree1 {
		changes = append(changes, treeChange{Path: entryPath, Change: "removed", Size: sig1.size})
	}

	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })

	out := compareSnapshotsOutput{Changes: changes}
	for _, change := range changes {
		switch change.Change {
		case "added":
			out.Added++
		case "removed":
			out.Removed++
		case "modified":
			out.Modified++
		}
	}

	if len(out.Changes) > maxListEntries {
		out.Changes = out.Changes[:maxListEntries]
		out.Truncated = true
	}

	return out, nil
}

func fileHistory(repo *repository.Repository, in fileHistoryInput) (fileHistoryOutput, error) {
	if in.Path == "" {
		return fileHistoryOutput{}, fmt.Errorf("path is required")
	}

	snapshotIDs, err := locate.LocateSnapshotIDs(repo, locate.NewDefaultLocateOptions())
	if err != nil {
		return fileHistoryOutput{}, fmt.Errorf("could not fetch snapshots list: %w", err)
	}

	// Oldest first, so that "changed" can be computed against the version that
	// came before; the output is reversed to newest first at the end. Sorting
	// needs the timestamps, so the headers are loaded once for ordering and the
	// snapshots revisited one at a time to keep only one open.
	byTime := make([]struct {
		id objects.MAC
		ts time.Time
	}, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			return fileHistoryOutput{}, fmt.Errorf("could not load snapshot: %w", err)
		}
		byTime = append(byTime, struct {
			id objects.MAC
			ts time.Time
		}{snapshotID, snap.Header.Timestamp})
		snap.Close()
	}
	sort.Slice(byTime, func(i, j int) bool { return byTime[i].ts.Before(byTime[j].ts) })

	out := fileHistoryOutput{Path: in.Path}
	versions := make([]fileVersion, 0)
	previous := ""
	for _, candidate := range byTime {
		snap, err := snapshot.Load(repo, candidate.id)
		if err != nil {
			return fileHistoryOutput{}, fmt.Errorf("could not load snapshot: %w", err)
		}

		version, found, err := fileVersionIn(snap, in.Path)
		snap.Close()
		if err != nil {
			return fileHistoryOutput{}, err
		}
		if !found {
			continue
		}

		if len(versions) == maxListEntries {
			out.Truncated = true
			break
		}

		version.Changed = version.Object != previous
		previous = version.Object
		versions = append(versions, version)
	}

	for i := len(versions) - 1; i >= 0; i-- {
		out.Versions = append(out.Versions, versions[i])
	}

	return out, nil
}

// fileVersionIn reports the version of pathname held by snap, if any.
func fileVersionIn(snap *snapshot.Snapshot, pathname string) (fileVersion, bool, error) {
	pvfs, err := snap.Filesystem()
	if err != nil {
		return fileVersion{}, false, err
	}

	entry, err := pvfs.GetEntry(snapshotAbsPath(pathname))
	if err != nil {
		// The file is simply absent from this snapshot.
		return fileVersion{}, false, nil
	}
	if entry.IsDir() {
		return fileVersion{}, false, fmt.Errorf("%s: is a directory", pathname)
	}

	object := hex.EncodeToString(entry.Object[:])
	if entry.FileInfo.Mode()&fs.ModeSymlink != 0 {
		object = "symlink:" + entry.SymlinkTarget
	}

	return fileVersion{
		Snapshot:     hex.EncodeToString(snap.Header.GetIndexShortID()),
		SnapshotTime: snap.Header.Timestamp.UTC().Format(time.RFC3339),
		Size:         entry.Stat().Size(),
		ModTime:      entry.Stat().ModTime().UTC().Format(time.RFC3339),
		Object:       object,
	}, true, nil
}

// snapshotAbsPath anchors a client-supplied path at the snapshot root.
func snapshotAbsPath(pathname string) string {
	if pathname == "" {
		return "/"
	}
	if pathname[0] != '/' {
		return "/" + pathname
	}
	return pathname
}
