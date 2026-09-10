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
// operator's restore root and guarantees the result cannot escape it.
func resolveRestoreDestination(root, destination string) (string, error) {
	if destination == "" {
		return root, nil
	}
	if filepath.IsAbs(destination) {
		return "", fmt.Errorf("destination must be relative to the restore root")
	}

	target := filepath.Join(root, filepath.Clean(destination))
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("destination escapes the restore root")
	}

	return target, nil
}

func restoreFiles(ctx *appcontext.AppContext, repo *repository.Repository, in restoreFilesInput, restoreRoot string) (restoreFilesOutput, error) {
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

	if err := os.MkdirAll(target, 0o755); err != nil {
		return restoreFilesOutput{}, err
	}

	exporterInstance, err := exporter.NewExporter(ctx.GetInner(), ctx.ExporterOpts(), map[string]string{
		"location": target,
	})
	if err != nil {
		return restoreFilesOutput{}, err
	}
	defer exporterInstance.Close(ctx)

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
