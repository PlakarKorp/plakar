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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
)

// restoreFilesInput mirrors "plakar restore", confined to the directory the
// operator named with -restore-root: a model picks what to restore, never
// where to.
type restoreFilesInput struct {
	Snapshot        string `json:"snapshot" jsonschema:"snapshot ID as returned by list_snapshots"`
	Path            string `json:"path,omitempty" jsonschema:"file or directory inside the snapshot to restore, the whole snapshot by default"`
	Destination     string `json:"destination,omitempty" jsonschema:"directory to restore into, relative to the restore root, the root itself by default"`
	SkipPermissions bool   `json:"skip_permissions,omitempty" jsonschema:"do not restore file permissions"`
}

type restoreFilesOutput struct {
	Snapshot    string `json:"snapshot"`
	Path        string `json:"path" jsonschema:"path inside the snapshot that was restored"`
	Destination string `json:"destination" jsonschema:"absolute directory the files were written under"`
}

// resolveRestoreDestination joins a client-supplied relative directory with the
// operator's restore root and guarantees the result cannot escape it, creating
// it in the process. The check cannot be lexical: snapshots contain symlinks
// and restores recreate them, so a name under the root may point anywhere.
// os.Root refuses to traverse a symlink that leaves the root.
func resolveRestoreDestination(root, destination string) (string, error) {
	if destination == "" {
		return root, nil
	}
	if filepath.IsAbs(destination) {
		return "", fmt.Errorf("destination must be relative to the restore root")
	}

	clean := filepath.Clean(destination)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("destination escapes the restore root")
	}

	anchored, err := os.OpenRoot(root)
	if err != nil {
		return "", fmt.Errorf("restore root: %w", err)
	}
	defer anchored.Close()

	if err := anchored.MkdirAll(clean, 0o755); err != nil {
		return "", fmt.Errorf("destination: %w", err)
	}

	return filepath.Join(root, clean), nil
}

func restoreFiles(ctx *appcontext.AppContext, repo *repository.Repository, in restoreFilesInput, restoreRoot string) (out restoreFilesOutput, err error) {
	if in.Snapshot == "" {
		return restoreFilesOutput{}, fmt.Errorf("snapshot is required")
	}

	target, err := resolveRestoreDestination(restoreRoot, in.Destination)
	if err != nil {
		return restoreFilesOutput{}, err
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath(in.Snapshot, in.Path))
	if err != nil {
		return restoreFilesOutput{}, err
	}
	defer snap.Close()

	exporterInstance, err := exporter.NewExporter(ctx.GetInner(), ctx.ExporterOpts(), map[string]string{
		"location": target,
	})
	if err != nil {
		return restoreFilesOutput{}, err
	}
	// The exporter is what writes the restored bytes, so a failed close means a
	// possibly incomplete restore and must not be reported as success.
	defer func() {
		if closeErr := exporterInstance.Close(ctx); closeErr != nil && err == nil {
			out, err = restoreFilesOutput{}, fmt.Errorf("closing exporter: %w", closeErr)
		}
	}()

	// Export strips the restored entry's own prefix: a restored file lands
	// directly in the destination, a restored directory spills its contents
	// there.
	options := &snapshot.ExportOptions{
		SkipPermissions: in.SkipPermissions,
	}

	if err := snap.Export(exporterInstance, pathname, options); err != nil {
		return restoreFilesOutput{}, err
	}

	return restoreFilesOutput{
		Snapshot:    in.Snapshot,
		Path:        pathname,
		Destination: target,
	}, nil
}
