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
	"io/fs"
	"path"
	"time"
	"unicode/utf8"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The tools below are deliberately read-only: an MCP client drives them on
// behalf of a model, and a model has no business mutating backups.

type listSnapshotsInput struct{}

type snapshotInfo struct {
	ID        string   `json:"id" jsonschema:"short ID of the snapshot, usable wherever a snapshot is expected"`
	Timestamp string   `json:"timestamp" jsonschema:"RFC3339 creation time of the snapshot"`
	Source    string   `json:"source" jsonschema:"directory the snapshot was taken from"`
	Size      uint64   `json:"size" jsonschema:"logical size of the snapshot in bytes"`
	Tags      []string `json:"tags,omitempty" jsonschema:"tags attached to the snapshot"`
}

type listSnapshotsOutput struct {
	Snapshots []snapshotInfo `json:"snapshots"`
}

type repositoryInfoInput struct{}

type repositoryInfoOutput struct {
	Snapshots   int    `json:"snapshots" jsonschema:"number of snapshots in the repository"`
	LogicalSize int64  `json:"logical_size" jsonschema:"total logical size of all snapshots in bytes"`
	StorageSize int64  `json:"storage_size" jsonschema:"bytes actually occupied in the store, after deduplication and compression"`
	Encrypted   bool   `json:"encrypted" jsonschema:"whether the repository is encrypted"`
	Compression string `json:"compression,omitempty" jsonschema:"compression algorithm, empty when disabled"`
	Hashing     string `json:"hashing" jsonschema:"hashing algorithm"`
	Chunking    string `json:"chunking" jsonschema:"chunking algorithm"`
}

type listFilesInput struct {
	Snapshot  string `json:"snapshot" jsonschema:"snapshot ID as returned by list_snapshots"`
	Path      string `json:"path,omitempty" jsonschema:"directory inside the snapshot to list, defaults to the snapshot root"`
	Recursive bool   `json:"recursive,omitempty" jsonschema:"list every entry below path instead of just its direct children"`
}

type fileEntry struct {
	Path    string `json:"path" jsonschema:"path of the entry inside the snapshot"`
	Name    string `json:"name" jsonschema:"base name of the entry"`
	Size    int64  `json:"size" jsonschema:"size of the entry in bytes"`
	Mode    string `json:"mode" jsonschema:"file mode, in the form drwxr-xr-x"`
	ModTime string `json:"mod_time" jsonschema:"RFC3339 modification time"`
	IsDir   bool   `json:"is_dir" jsonschema:"whether the entry is a directory"`
}

type listFilesOutput struct {
	Entries   []fileEntry `json:"entries"`
	Truncated bool        `json:"truncated" jsonschema:"true when the listing was cut short because it was too large"`
}

type readFileInput struct {
	Snapshot string `json:"snapshot" jsonschema:"snapshot ID as returned by list_snapshots"`
	Path     string `json:"path" jsonschema:"path of the file to read inside the snapshot"`
}

type readFileOutput struct {
	Path      string `json:"path" jsonschema:"path of the file that was read"`
	Size      int64  `json:"size" jsonschema:"full size of the file in bytes"`
	Content   string `json:"content" jsonschema:"content of the file, empty when the file is not valid UTF-8 text"`
	Binary    bool   `json:"binary" jsonschema:"true when the file is not valid UTF-8 text, in which case content is empty"`
	Truncated bool   `json:"truncated" jsonschema:"true when only the first max-file-size bytes were returned"`
}

// maxListEntries bounds a listing so that a recursive walk of a large snapshot
// cannot produce a response no client can consume.
const maxListEntries = 10000

func (cmd *Mcp) registerTools(ctx *appcontext.AppContext, repo *repository.Repository, server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_snapshots",
		Description: "List the snapshots stored in the Plakar repository, most useful as a first step to discover snapshot IDs.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ listSnapshotsInput) (*mcp.CallToolResult, listSnapshotsOutput, error) {
		out, err := listSnapshots(repo)
		if err != nil {
			return nil, listSnapshotsOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "repository_info",
		Description: "Report configuration and size information about the Plakar repository.",
	}, func(callCtx context.Context, _ *mcp.CallToolRequest, _ repositoryInfoInput) (*mcp.CallToolResult, repositoryInfoOutput, error) {
		out, err := repositoryInfo(callCtx, repo)
		if err != nil {
			return nil, repositoryInfoOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_files",
		Description: "List the files and directories contained in a snapshot.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in listFilesInput) (*mcp.CallToolResult, listFilesOutput, error) {
		out, err := listFiles(repo, in)
		if err != nil {
			return nil, listFilesOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_file",
		Description: "Read the contents of a regular file stored in a snapshot. Binary files are reported as such rather than returned.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in readFileInput) (*mcp.CallToolResult, readFileOutput, error) {
		out, err := readFile(repo, in, cmd.MaxFileSize)
		if err != nil {
			return nil, readFileOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "snapshot_details",
		Description: "Report the full metadata of a snapshot: when and where it was taken, by whom, and what it contains.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in snapshotDetailsInput) (*mcp.CallToolResult, snapshotDetailsOutput, error) {
		out, err := snapshotDetails(repo, in)
		if err != nil {
			return nil, snapshotDetailsOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_files",
		Description: "Search for files across one or every snapshot, by name pattern and/or MIME type. The way to answer questions like which snapshot contains a given file.",
	}, func(callCtx context.Context, _ *mcp.CallToolRequest, in searchFilesInput) (*mcp.CallToolResult, searchFilesOutput, error) {
		out, err := searchFiles(callCtx, repo, in)
		if err != nil {
			return nil, searchFilesOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stat_entry",
		Description: "Report the detailed metadata of a single file or directory in a snapshot: ownership, mode, content type, extended attributes and classifications.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in statEntryInput) (*mcp.CallToolResult, statEntryOutput, error) {
		out, err := statEntry(repo, in)
		if err != nil {
			return nil, statEntryOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "diff_snapshots",
		Description: "Compare the same file in two snapshots and return a unified diff. Only compares snapshots against each other, never against the local filesystem.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in diffSnapshotsInput) (*mcp.CallToolResult, diffSnapshotsOutput, error) {
		out, err := diffSnapshots(repo, in, cmd.MaxFileSize)
		if err != nil {
			return nil, diffSnapshotsOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "digest_file",
		Description: "Compute the cryptographic digest of a file stored in a snapshot, to compare it against a known hash.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in digestFileInput) (*mcp.CallToolResult, digestFileOutput, error) {
		out, err := digestFile(repo, in)
		if err != nil {
			return nil, digestFileOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_snapshot",
		Description: "Verify the integrity of a snapshot. Checks metadata only by default; set deep to re-read and rehash every chunk, which is accurate but slow on large snapshots.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in checkSnapshotInput) (*mcp.CallToolResult, checkSnapshotOutput, error) {
		out, err := checkSnapshot(ctx, repo, in)
		if err != nil {
			return nil, checkSnapshotOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "snapshot_errors",
		Description: "List the entries that could not be read when a snapshot was created, to tell an incomplete backup from a complete one.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in snapshotErrorsInput) (*mcp.CallToolResult, snapshotErrorsOutput, error) {
		out, err := snapshotErrors(repo, in)
		if err != nil {
			return nil, snapshotErrorsOutput{}, err
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "repository_locks",
		Description: "List the locks currently held on the repository, to tell whether another Plakar process is operating on it.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ repositoryLocksInput) (*mcp.CallToolResult, repositoryLocksOutput, error) {
		out, err := repositoryLocks(repo)
		if err != nil {
			return nil, repositoryLocksOutput{}, err
		}
		return nil, out, nil
	})

	// The write tools only exist when they have been explicitly enabled: a
	// client cannot discover, let alone call, what was not registered.
	if cmd.AllowBackup {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "create_backup",
			Description: "Back up a directory into the repository, creating a new snapshot. Reads the source from the local filesystem.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create a backup",
				ReadOnlyHint:    false,
				DestructiveHint: ptr(false),
			},
		}, func(_ context.Context, _ *mcp.CallToolRequest, in createBackupInput) (*mcp.CallToolResult, createBackupOutput, error) {
			out, err := createBackup(ctx, repo, in)
			if err != nil {
				return nil, createBackupOutput{}, err
			}
			return nil, out, nil
		})
	}

	if cmd.AllowDelete {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "remove_snapshots",
			Description: "Permanently remove the named snapshots from the repository. Reports what would be removed unless apply is set. This cannot be undone.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Remove snapshots",
				ReadOnlyHint:    false,
				DestructiveHint: ptr(true),
			},
		}, func(_ context.Context, _ *mcp.CallToolRequest, in removeSnapshotsInput) (*mcp.CallToolResult, removeSnapshotsOutput, error) {
			out, err := removeSnapshots(ctx, repo, in)
			if err != nil {
				return nil, removeSnapshotsOutput{}, err
			}
			return nil, out, nil
		})

		mcp.AddTool(server, &mcp.Tool{
			Name:        "prune_snapshots",
			Description: "Permanently remove the snapshots the repository retention policy selects. Reports what would be removed unless apply is set. This cannot be undone.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Prune snapshots",
				ReadOnlyHint:    false,
				DestructiveHint: ptr(true),
			},
		}, func(_ context.Context, _ *mcp.CallToolRequest, in pruneSnapshotsInput) (*mcp.CallToolResult, pruneSnapshotsOutput, error) {
			out, err := pruneSnapshots(ctx, repo, in)
			if err != nil {
				return nil, pruneSnapshotsOutput{}, err
			}
			return nil, out, nil
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func listSnapshots(repo *repository.Repository) (listSnapshotsOutput, error) {
	snapshotIDs, err := locate.LocateSnapshotIDs(repo, locate.NewDefaultLocateOptions())
	if err != nil {
		return listSnapshotsOutput{}, fmt.Errorf("could not fetch snapshots list: %w", err)
	}

	out := listSnapshotsOutput{Snapshots: make([]snapshotInfo, 0, len(snapshotIDs))}
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			return listSnapshotsOutput{}, fmt.Errorf("could not fetch snapshot: %w", err)
		}

		source := snap.Header.GetSource(0)
		out.Snapshots = append(out.Snapshots, snapshotInfo{
			ID:        hex.EncodeToString(snap.Header.GetIndexShortID()),
			Timestamp: snap.Header.Timestamp.UTC().Format(time.RFC3339),
			Source:    source.Importer.Directory,
			Size:      source.Summary.Directory.Size + source.Summary.Below.Size,
			Tags:      snap.Header.Tags,
		})

		snap.Close()
	}

	return out, nil
}

func repositoryInfo(ctx context.Context, repo *repository.Repository) (repositoryInfoOutput, error) {
	configuration := repo.Configuration()

	nSnapshots, logicalSize, err := snapshot.LogicalSize(repo)
	if err != nil {
		return repositoryInfoOutput{}, fmt.Errorf("unable to calculate logical size: %w", err)
	}

	storageSize, err := repo.Store().Size(ctx)
	if err != nil {
		return repositoryInfoOutput{}, fmt.Errorf("unable to compute storage size: %w", err)
	}

	out := repositoryInfoOutput{
		Snapshots:   nSnapshots,
		LogicalSize: logicalSize,
		StorageSize: storageSize,
		Encrypted:   configuration.Encryption != nil,
		Hashing:     configuration.Hashing.Algorithm,
		Chunking:    configuration.Chunking.Algorithm,
	}
	if configuration.Compression != nil {
		out.Compression = configuration.Compression.Algorithm
	}

	return out, nil
}

func listFiles(repo *repository.Repository, in listFilesInput) (listFilesOutput, error) {
	if in.Snapshot == "" {
		return listFilesOutput{}, fmt.Errorf("snapshot is required")
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath(in.Snapshot, in.Path))
	if err != nil {
		return listFilesOutput{}, err
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		return listFilesOutput{}, err
	}

	out := listFilesOutput{Entries: make([]fileEntry, 0)}

	resolved := false
	err = pvfs.WalkDir(pathname, func(entryPath string, d *vfs.Entry, err error) error {
		if err != nil {
			return err
		}
		if !resolved {
			// pathname may point at a symlink, so anchor the walk on the
			// physical path the VFS resolved it to.
			resolved = true
			pathname = d.Path()
		}
		if d.IsDir() && entryPath == pathname {
			return nil
		}

		sb, err := d.Info()
		if err != nil {
			return err
		}

		if len(out.Entries) == maxListEntries {
			out.Truncated = true
			return fs.SkipAll
		}

		out.Entries = append(out.Entries, fileEntry{
			Path:    entryPath,
			Name:    d.Name(),
			Size:    sb.Size(),
			Mode:    sb.Mode().String(),
			ModTime: sb.ModTime().UTC().Format(time.RFC3339),
			IsDir:   d.IsDir(),
		})

		if d.IsDir() && !in.Recursive && entryPath != pathname {
			return fs.SkipDir
		}

		return nil
	})
	if err != nil {
		return listFilesOutput{}, err
	}

	return out, nil
}

func readFile(repo *repository.Repository, in readFileInput, maxFileSize int64) (readFileOutput, error) {
	if in.Snapshot == "" {
		return readFileOutput{}, fmt.Errorf("snapshot is required")
	}
	if in.Path == "" {
		return readFileOutput{}, fmt.Errorf("path is required")
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath(in.Snapshot, in.Path))
	if err != nil {
		return readFileOutput{}, err
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		return readFileOutput{}, err
	}

	entry, err := pvfs.GetEntry(pathname)
	if err != nil {
		return readFileOutput{}, fmt.Errorf("%s: no such file", in.Path)
	}

	if !entry.Stat().Mode().IsRegular() {
		return readFileOutput{}, fmt.Errorf("%s: not a regular file", in.Path)
	}

	file, err := entry.Open(pvfs)
	if err != nil {
		return readFileOutput{}, err
	}
	defer file.Close()

	size := entry.Stat().Size()
	out := readFileOutput{Path: pathname, Size: size}

	// Read one byte past the limit so a file sitting exactly on it is not
	// reported as truncated.
	data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	if err != nil {
		return readFileOutput{}, err
	}
	if int64(len(data)) > maxFileSize {
		data = data[:maxFileSize]
		out.Truncated = true
	}

	// A model can do nothing with raw bytes, and invalid UTF-8 would not
	// survive JSON encoding intact, so binary files are flagged instead.
	if !utf8.Valid(data) {
		out.Binary = true
		return out, nil
	}

	out.Content = string(data)

	return out, nil
}

// snapshotPath builds the SNAPSHOT[:PATH] form the locate package expects.
func snapshotPath(snapshotID, pathname string) string {
	if pathname == "" {
		pathname = "/"
	}
	if !path.IsAbs(pathname) {
		pathname = "/" + pathname
	}
	return snapshotID + ":" + pathname
}
