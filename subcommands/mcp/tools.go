/*
 * Copyright (c) 2026 Gilles Chehade <gilles@poolp.org>
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
	"path"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools adds the plakar tool, which reads, and the write tools the
// flags enable. The write tools only exist when they have been explicitly
// enabled: a client cannot discover, let alone call, what was not registered.
func (cmd *Mcp) registerTools(ctx *appcontext.AppContext, repo *repository.Repository, server *mcp.Server) {
	cmd.registerCLI(ctx, repo, server)

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

	if cmd.AllowRestore {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "restore_files",
			Description: "Restore a file or directory from a snapshot into the directory the server was started with; destinations outside it are refused.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Restore files",
				ReadOnlyHint:    false,
				DestructiveHint: ptr(true),
			},
		}, func(_ context.Context, _ *mcp.CallToolRequest, in restoreFilesInput) (*mcp.CallToolResult, restoreFilesOutput, error) {
			out, err := restoreFiles(ctx, repo, in, cmd.RestoreRoot)
			if err != nil {
				return nil, restoreFilesOutput{}, err
			}
			return nil, out, nil
		})
	}

	if cmd.AllowSync {
		mcp.AddTool(server, &mcp.Tool{
			Name:        "sync_snapshots",
			Description: "Push snapshots to a peer repository named in the Plakar configuration. Only pushes: the served repository is never written to.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Sync snapshots to a peer",
				ReadOnlyHint:    false,
				DestructiveHint: ptr(false),
			},
		}, func(_ context.Context, _ *mcp.CallToolRequest, in syncSnapshotsInput) (*mcp.CallToolResult, syncSnapshotsOutput, error) {
			out, err := syncSnapshots(ctx, repo, in)
			if err != nil {
				return nil, syncSnapshotsOutput{}, err
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
