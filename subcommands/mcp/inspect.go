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
	"io"
	"path"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/pmezard/go-difflib/difflib"
)

// snapshotDetailsInput mirrors "plakar info SNAPSHOT".
type snapshotDetailsInput struct {
	Snapshot string `json:"snapshot" jsonschema:"snapshot ID as returned by list_snapshots"`
}

type snapshotSummary struct {
	Directories uint64 `json:"directories"`
	Files       uint64 `json:"files"`
	Symlinks    uint64 `json:"symlinks"`
	Objects     uint64 `json:"objects"`
	Chunks      uint64 `json:"chunks"`
	Size        uint64 `json:"size" jsonschema:"total size of the snapshot contents in bytes"`
	Errors      uint64 `json:"errors" jsonschema:"number of entries that could not be read when the snapshot was created"`
}

type snapshotDetailsOutput struct {
	ID          string          `json:"id" jsonschema:"full snapshot ID"`
	ShortID     string          `json:"short_id" jsonschema:"short snapshot ID"`
	Timestamp   string          `json:"timestamp" jsonschema:"RFC3339 creation time"`
	Duration    string          `json:"duration" jsonschema:"how long the snapshot took to create"`
	Name        string          `json:"name,omitempty"`
	Category    string          `json:"category,omitempty"`
	Environment string          `json:"environment,omitempty"`
	Perimeter   string          `json:"perimeter,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Root        string          `json:"root" jsonschema:"MAC of the snapshot filesystem root"`
	Importer    string          `json:"importer" jsonschema:"importer type used to create the snapshot"`
	Origin      string          `json:"origin" jsonschema:"origin the snapshot was taken from"`
	Directory   string          `json:"directory" jsonschema:"source directory of the snapshot"`
	Hostname    string          `json:"hostname,omitempty"`
	Username    string          `json:"username,omitempty"`
	OS          string          `json:"operating_system,omitempty"`
	Arch        string          `json:"architecture,omitempty"`
	Summary     snapshotSummary `json:"summary"`
}

// searchFilesInput mirrors "plakar locate" and "plakar diag search".
type searchFilesInput struct {
	Snapshot string   `json:"snapshot,omitempty" jsonschema:"snapshot to search in, all snapshots are searched when empty"`
	Name     string   `json:"name,omitempty" jsonschema:"substring or glob to match against file names, for example *.pdf"`
	Path     string   `json:"path,omitempty" jsonschema:"restrict the search to this directory prefix"`
	Mimes    []string `json:"mimes,omitempty" jsonschema:"restrict results to these MIME types, for example text/plain"`
	Limit    int      `json:"limit,omitempty" jsonschema:"maximum number of matches to return, defaults to 1000"`
	Offset   int      `json:"offset,omitempty" jsonschema:"number of matches to skip, for paging through a truncated search"`
}

type searchMatch struct {
	Snapshot string `json:"snapshot" jsonschema:"short ID of the snapshot the match was found in"`
	Path     string `json:"path" jsonschema:"path of the matching entry"`
	Size     int64  `json:"size"`
	IsDir    bool   `json:"is_dir"`
}

type searchFilesOutput struct {
	Matches    []searchMatch `json:"matches"`
	Truncated  bool          `json:"truncated" jsonschema:"true when more matches existed than limit allowed"`
	NextOffset int           `json:"next_offset,omitempty" jsonschema:"offset to pass to fetch the next page, present when truncated"`
}

// diffSnapshotsInput mirrors "plakar diff", restricted to comparing two
// snapshots: the CLI can also diff against the live filesystem, which an MCP
// client has no business reaching.
type diffSnapshotsInput struct {
	Snapshot1 string `json:"snapshot1" jsonschema:"first snapshot ID"`
	Snapshot2 string `json:"snapshot2" jsonschema:"second snapshot ID"`
	Path      string `json:"path" jsonschema:"path of the file to compare in both snapshots"`
}

type diffSnapshotsOutput struct {
	Path      string `json:"path"`
	Identical bool   `json:"identical" jsonschema:"true when the file is byte-identical in both snapshots"`
	Diff      string `json:"diff,omitempty" jsonschema:"unified diff, empty when the files are identical or binary"`
	Binary    bool   `json:"binary,omitempty" jsonschema:"true when either side is not valid UTF-8 text, in which case no diff is produced"`
	Truncated bool   `json:"truncated,omitempty" jsonschema:"true when either side was truncated before diffing"`
}

// digestFileInput mirrors "plakar digest".
type digestFileInput struct {
	Snapshot  string `json:"snapshot" jsonschema:"snapshot ID as returned by list_snapshots"`
	Path      string `json:"path" jsonschema:"path of the file to hash inside the snapshot"`
	Algorithm string `json:"algorithm,omitempty" jsonschema:"hashing algorithm, for example SHA256 or BLAKE3, defaults to SHA256"`
}

type digestFileOutput struct {
	Path      string `json:"path"`
	Algorithm string `json:"algorithm"`
	Digest    string `json:"digest" jsonschema:"hex-encoded digest of the full file contents"`
}

// checkSnapshotInput mirrors "plakar check".
type checkSnapshotInput struct {
	Snapshot string `json:"snapshot" jsonschema:"snapshot ID as returned by list_snapshots"`
	Path     string `json:"path,omitempty" jsonschema:"restrict the check to this path inside the snapshot"`
	Deep     bool   `json:"deep,omitempty" jsonschema:"re-read and rehash every chunk instead of only checking metadata; accurate but slow and I/O heavy on large snapshots"`
}

type checkSnapshotOutput struct {
	Snapshot string `json:"snapshot"`
	Deep     bool   `json:"deep" jsonschema:"whether every chunk was rehashed"`
	OK       bool   `json:"ok" jsonschema:"true when no integrity problem was found"`
	Error    string `json:"error,omitempty" jsonschema:"description of the integrity problem when ok is false"`
}

// snapshotErrorsInput mirrors "plakar info -errors SNAPSHOT".
type snapshotErrorsInput struct {
	Snapshot string `json:"snapshot" jsonschema:"snapshot ID as returned by list_snapshots"`
	Path     string `json:"path,omitempty" jsonschema:"restrict the listing to this path inside the snapshot"`
}

type snapshotError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type snapshotErrorsOutput struct {
	Errors    []snapshotError `json:"errors"`
	Truncated bool            `json:"truncated"`
}

// repositoryLocksInput mirrors "plakar diag locks".
type repositoryLocksInput struct{}

type repositoryLock struct {
	ID        string `json:"id"`
	Exclusive bool   `json:"exclusive" jsonschema:"true for an exclusive lock, false for a shared one"`
	Timestamp string `json:"timestamp" jsonschema:"RFC3339 time the lock was taken"`
	Hostname  string `json:"hostname" jsonschema:"host holding the lock"`
}

type repositoryLocksOutput struct {
	Locks []repositoryLock `json:"locks"`
}

// statEntryInput mirrors "plakar diag vfs".
type statEntryInput struct {
	Snapshot string `json:"snapshot" jsonschema:"snapshot ID as returned by list_snapshots"`
	Path     string `json:"path" jsonschema:"path of the entry to stat inside the snapshot"`
}

type statEntryOutput struct {
	Path            string              `json:"path"`
	Name            string              `json:"name"`
	Size            int64               `json:"size"`
	Mode            string              `json:"mode"`
	ModTime         string              `json:"mod_time"`
	IsDir           bool                `json:"is_dir"`
	Uid             uint64              `json:"uid"`
	Gid             uint64              `json:"gid"`
	Username        string              `json:"username,omitempty"`
	Groupname       string              `json:"groupname,omitempty"`
	NumLinks        uint16              `json:"num_links"`
	SymlinkTarget   string              `json:"symlink_target,omitempty"`
	ContentType     string              `json:"content_type,omitempty" jsonschema:"detected MIME type, for regular files"`
	Entropy         float64             `json:"entropy,omitempty" jsonschema:"Shannon entropy of the file contents"`
	Xattrs          []string            `json:"xattrs,omitempty" jsonschema:"names of the extended attributes attached to the entry"`
	Classifications map[string][]string `json:"classifications,omitempty" jsonschema:"analyzer name to classes assigned to the entry"`
	Tags            []string            `json:"tags,omitempty"`
}

const defaultSearchLimit = 1000

func statEntry(repo *repository.Repository, in statEntryInput) (statEntryOutput, error) {
	if in.Snapshot == "" {
		return statEntryOutput{}, fmt.Errorf("snapshot is required")
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath(in.Snapshot, in.Path))
	if err != nil {
		return statEntryOutput{}, err
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		return statEntryOutput{}, err
	}

	entry, err := pvfs.GetEntry(path.Clean(pathname))
	if err != nil {
		return statEntryOutput{}, fmt.Errorf("%s: no such entry", in.Path)
	}

	stat := entry.Stat()
	out := statEntryOutput{
		Path:          entry.Path(),
		Name:          stat.Name(),
		Size:          stat.Size(),
		Mode:          stat.Mode().String(),
		ModTime:       stat.ModTime().UTC().Format(time.RFC3339),
		IsDir:         stat.Mode().IsDir(),
		Uid:           stat.Uid(),
		Gid:           stat.Gid(),
		Username:      stat.Username(),
		Groupname:     stat.Groupname(),
		NumLinks:      stat.Nlink(),
		SymlinkTarget: entry.SymlinkTarget,
		Xattrs:        entry.ExtendedAttributes,
		Tags:          entry.Tags,
	}

	if entry.ResolvedObject != nil {
		out.ContentType = entry.ResolvedObject.ContentType
		out.Entropy = entry.ResolvedObject.Entropy
	}

	if len(entry.Classifications) > 0 {
		out.Classifications = make(map[string][]string, len(entry.Classifications))
		for _, classification := range entry.Classifications {
			out.Classifications[classification.Analyzer] = classification.Classes
		}
	}

	return out, nil
}

func snapshotDetails(repo *repository.Repository, in snapshotDetailsInput) (snapshotDetailsOutput, error) {
	if in.Snapshot == "" {
		return snapshotDetailsOutput{}, fmt.Errorf("snapshot is required")
	}

	snap, _, err := locate.OpenSnapshotByPath(repo, in.Snapshot+":")
	if err != nil {
		return snapshotDetailsOutput{}, err
	}
	defer snap.Close()

	header := snap.Header
	indexID := header.GetIndexID()
	source := header.GetSource(0)

	// Directory holds the counts for the snapshot root, Below the ones for
	// everything underneath it, so the totals are the sum of both.
	directory, below := source.Summary.Directory, source.Summary.Below

	return snapshotDetailsOutput{
		ID:          hex.EncodeToString(indexID[:]),
		ShortID:     hex.EncodeToString(header.GetIndexShortID()),
		Timestamp:   header.Timestamp.UTC().Format(time.RFC3339),
		Duration:    header.Duration.String(),
		Name:        header.Name,
		Category:    header.Category,
		Environment: header.Environment,
		Perimeter:   header.Perimeter,
		Tags:        header.Tags,
		Root:        fmt.Sprintf("%x", source.VFS.Root),
		Importer:    source.Importer.Type,
		Origin:      source.Importer.Origin,
		Directory:   source.Importer.Directory,
		Hostname:    header.GetContext("Hostname"),
		Username:    header.GetContext("Username"),
		OS:          header.GetContext("OperatingSystem"),
		Arch:        header.GetContext("Architecture"),
		Summary: snapshotSummary{
			Directories: directory.Directories + below.Directories,
			Files:       directory.Files + below.Files,
			Symlinks:    directory.Symlinks + below.Symlinks,
			Objects:     directory.Objects + below.Objects,
			Chunks:      directory.Chunks + below.Chunks,
			Size:        directory.Size + below.Size,
			Errors:      directory.Errors + below.Errors,
		},
	}, nil
}

func searchFiles(ctx context.Context, repo *repository.Repository, in searchFilesInput) (searchFilesOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}

	var snapshotIDs []objects.MAC
	if in.Snapshot == "" {
		ids, err := locate.LocateSnapshotIDs(repo, locate.NewDefaultLocateOptions())
		if err != nil {
			return searchFilesOutput{}, fmt.Errorf("could not fetch snapshots list: %w", err)
		}
		snapshotIDs = ids
	} else {
		snapshotIDs = locate.LookupSnapshotByPrefix(repo, in.Snapshot)
		if len(snapshotIDs) == 0 {
			return searchFilesOutput{}, fmt.Errorf("%s: no such snapshot", in.Snapshot)
		}
	}

	prefix := in.Path
	if prefix == "" {
		prefix = "/"
	}

	out := searchFilesOutput{Matches: make([]searchMatch, 0)}

	skipped := 0
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			return searchFilesOutput{}, fmt.Errorf("could not load snapshot: %w", err)
		}

		shortID := hex.EncodeToString(snap.Header.GetIndexShortID())

		it, err := snap.Search(ctx, &snapshot.SearchOpts{
			Recursive:  true,
			Prefix:     prefix,
			Mimes:      in.Mimes,
			NameFilter: in.Name,
		})
		if err != nil {
			snap.Close()
			return searchFilesOutput{}, err
		}

		for entry, err := range it {
			if err != nil {
				snap.Close()
				return searchFilesOutput{}, err
			}

			if skipped < in.Offset {
				skipped++
				continue
			}

			if len(out.Matches) == limit {
				out.Truncated = true
				out.NextOffset = in.Offset + limit
				break
			}

			out.Matches = append(out.Matches, searchMatch{
				Snapshot: shortID,
				Path:     entry.Path(),
				Size:     entry.Stat().Size(),
				IsDir:    entry.Stat().Mode().IsDir(),
			})
		}

		snap.Close()

		if out.Truncated {
			break
		}
	}

	return out, nil
}

func diffSnapshots(repo *repository.Repository, in diffSnapshotsInput, maxFileSize int64) (diffSnapshotsOutput, error) {
	if in.Snapshot1 == "" || in.Snapshot2 == "" {
		return diffSnapshotsOutput{}, fmt.Errorf("snapshot1 and snapshot2 are required")
	}
	if in.Path == "" {
		return diffSnapshotsOutput{}, fmt.Errorf("path is required")
	}

	first, err := readFile(repo, readFileInput{Snapshot: in.Snapshot1, Path: in.Path}, maxFileSize)
	if err != nil {
		return diffSnapshotsOutput{}, fmt.Errorf("snapshot1: %w", err)
	}

	second, err := readFile(repo, readFileInput{Snapshot: in.Snapshot2, Path: in.Path}, maxFileSize)
	if err != nil {
		return diffSnapshotsOutput{}, fmt.Errorf("snapshot2: %w", err)
	}

	out := diffSnapshotsOutput{
		Path:      in.Path,
		Truncated: first.Truncated || second.Truncated,
	}

	if first.Binary || second.Binary {
		out.Binary = true
		out.Identical = first.Size == second.Size && first.Content == second.Content
		return out, nil
	}

	if first.Content == second.Content {
		out.Identical = true
		return out, nil
	}

	unified, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(first.Content),
		B:        difflib.SplitLines(second.Content),
		FromFile: in.Snapshot1 + ":" + in.Path,
		ToFile:   in.Snapshot2 + ":" + in.Path,
		Context:  3,
	})
	if err != nil {
		return diffSnapshotsOutput{}, err
	}

	out.Diff = unified

	return out, nil
}

func digestFile(repo *repository.Repository, in digestFileInput) (digestFileOutput, error) {
	if in.Snapshot == "" {
		return digestFileOutput{}, fmt.Errorf("snapshot is required")
	}
	if in.Path == "" {
		return digestFileOutput{}, fmt.Errorf("path is required")
	}

	algorithm := strings.ToUpper(in.Algorithm)
	if algorithm == "" {
		algorithm = "SHA256"
	}
	hasher := hashing.GetHasher(algorithm)
	if hasher == nil {
		return digestFileOutput{}, fmt.Errorf("unsupported hashing algorithm: %s", algorithm)
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath(in.Snapshot, in.Path))
	if err != nil {
		return digestFileOutput{}, err
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		return digestFileOutput{}, err
	}

	entry, err := pvfs.GetEntry(pathname)
	if err != nil {
		return digestFileOutput{}, fmt.Errorf("%s: no such file", in.Path)
	}
	if !entry.Stat().Mode().IsRegular() {
		return digestFileOutput{}, fmt.Errorf("%s: not a regular file", in.Path)
	}

	rd, err := snap.NewReader(pathname)
	if err != nil {
		return digestFileOutput{}, err
	}
	defer rd.Close()

	if _, err := io.Copy(hasher, rd); err != nil {
		return digestFileOutput{}, err
	}

	return digestFileOutput{
		Path:      pathname,
		Algorithm: algorithm,
		Digest:    hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}

func checkSnapshot(ctx *appcontext.AppContext, repo *repository.Repository, in checkSnapshotInput) (checkSnapshotOutput, error) {
	if in.Snapshot == "" {
		return checkSnapshotOutput{}, fmt.Errorf("snapshot is required")
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath(in.Snapshot, in.Path))
	if err != nil {
		return checkSnapshotOutput{}, err
	}
	defer snap.Close()

	checkCache, err := ctx.GetCache().Check()
	if err != nil {
		return checkSnapshotOutput{}, err
	}
	defer checkCache.Close()

	snap.SetCheckCache(checkCache)

	out := checkSnapshotOutput{
		Snapshot: hex.EncodeToString(snap.Header.GetIndexShortID()),
		Deep:     in.Deep,
	}

	// FastCheck skips re-reading chunk payloads, so it is the default here: a
	// deep check on a large snapshot is minutes-to-hours of I/O.
	if err := snap.Check(pathname, &snapshot.CheckOptions{FastCheck: !in.Deep}); err != nil {
		out.OK = false
		out.Error = err.Error()
		return out, nil
	}

	out.OK = true

	return out, nil
}

func snapshotErrors(repo *repository.Repository, in snapshotErrorsInput) (snapshotErrorsOutput, error) {
	if in.Snapshot == "" {
		return snapshotErrorsOutput{}, fmt.Errorf("snapshot is required")
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath(in.Snapshot, in.Path))
	if err != nil {
		return snapshotErrorsOutput{}, err
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		return snapshotErrorsOutput{}, err
	}

	out := snapshotErrorsOutput{Errors: make([]snapshotError, 0)}
	for item, err := range pvfs.Errors(pathname) {
		if err != nil {
			return snapshotErrorsOutput{}, fmt.Errorf("failed to scan errors: %w", err)
		}
		if len(out.Errors) == maxListEntries {
			out.Truncated = true
			break
		}
		out.Errors = append(out.Errors, snapshotError{Path: item.Name, Error: item.Error})
	}

	return out, nil
}

func repositoryLocks(repo *repository.Repository) (repositoryLocksOutput, error) {
	lockIDs, err := repo.GetLocks()
	if err != nil {
		return repositoryLocksOutput{}, err
	}

	out := repositoryLocksOutput{Locks: make([]repositoryLock, 0, len(lockIDs))}
	for _, lockID := range lockIDs {
		rd, err := repo.GetLock(lockID)
		if err != nil {
			// A lock can disappear while we enumerate: it was released.
			continue
		}

		lock, err := repository.NewLockFromStream(rd)
		rd.Close()
		if err != nil {
			continue
		}

		out.Locks = append(out.Locks, repositoryLock{
			ID:        fmt.Sprintf("%x", lockID),
			Exclusive: lock.Exclusive,
			Timestamp: lock.Timestamp.UTC().Format(time.RFC3339),
			Hostname:  lock.Hostname,
		})
	}

	return out, nil
}
