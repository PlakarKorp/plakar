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
	"sync"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands/backup"
)

// createBackupInput mirrors "plakar backup".
type createBackupInput struct {
	Source string   `json:"source" jsonschema:"directory to back up, or @name to use a source from the Plakar configuration"`
	Name   string   `json:"name,omitempty" jsonschema:"name to record on the snapshot"`
	Tags   []string `json:"tags,omitempty" jsonschema:"tags to attach to the snapshot"`
	DryRun bool     `json:"dry_run,omitempty" jsonschema:"scan the source and report what would be backed up without writing a snapshot"`
}

type createBackupOutput struct {
	Snapshot string `json:"snapshot,omitempty" jsonschema:"short ID of the snapshot that was created, empty on a dry run"`
	Source   string `json:"source" jsonschema:"source that was backed up"`
	DryRun   bool   `json:"dry_run" jsonschema:"whether this was a dry run, in which case nothing was written"`
}

// removeSnapshotsInput mirrors "plakar rm", restricted to explicit IDs: a
// filter-based removal driven by a model is too easy to get catastrophically
// wrong, so the caller has to name what it wants gone.
type removeSnapshotsInput struct {
	Snapshots []string `json:"snapshots" jsonschema:"IDs of the snapshots to remove, as returned by list_snapshots"`
	Apply     bool     `json:"apply,omitempty" jsonschema:"actually remove the snapshots; when false, only report what would be removed"`
}

type removedSnapshot struct {
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
}

type removeSnapshotsOutput struct {
	Snapshots []removedSnapshot `json:"snapshots" jsonschema:"the snapshots that were removed, or would be removed on a dry run"`
	Applied   bool              `json:"applied" jsonschema:"true when the snapshots were actually removed"`
}

// pruneSnapshotsInput mirrors "plakar prune": the retention policy comes from
// the repository configuration, so there is nothing to pass but the switch that
// makes it real.
type pruneSnapshotsInput struct {
	Apply bool `json:"apply,omitempty" jsonschema:"actually remove the snapshots the retention policy selects; when false, only report them"`
}

type pruneSnapshotsOutput struct {
	Snapshots []removedSnapshot `json:"snapshots" jsonschema:"the snapshots the policy selected for removal"`
	Applied   bool              `json:"applied" jsonschema:"true when the snapshots were actually removed"`
}

func createBackup(ctx *appcontext.AppContext, repo *repository.Repository, in createBackupInput) (createBackupOutput, error) {
	if in.Source == "" {
		return createBackupOutput{}, fmt.Errorf("source is required")
	}

	// Reuse the backup subcommand rather than reimplementing it: importer
	// resolution, exclude rules and hooks all live there.
	cmd := &backup.Backup{
		Sources: []string{in.Source},
		Name:    in.Name,
		Tags:    in.Tags,
		DryRun:  in.DryRun,
		Opts:    make(map[string]string),
		// The progress importer duplicates the scan to report statistics, which
		// is noise for a programmatic caller.
		NoProgress: true,
	}
	cmd.RepositorySecret = ctx.GetSecret()

	status, err, snapshotID, hookErr := cmd.DoBackup(ctx, repo)
	if err != nil {
		return createBackupOutput{}, err
	}
	if hookErr != nil {
		return createBackupOutput{}, hookErr
	}
	if status != 0 {
		return createBackupOutput{}, fmt.Errorf("backup failed")
	}

	out := createBackupOutput{Source: in.Source, DryRun: in.DryRun}
	if !in.DryRun {
		// The snapshot MAC is the header identifier, so its first four bytes
		// are the short ID the other tools speak.
		out.Snapshot = hex.EncodeToString(snapshotID[:4])
	}

	return out, nil
}

func removeSnapshots(ctx *appcontext.AppContext, repo *repository.Repository, in removeSnapshotsInput) (removeSnapshotsOutput, error) {
	if len(in.Snapshots) == 0 {
		return removeSnapshotsOutput{}, fmt.Errorf("snapshots is required")
	}

	options := locate.NewDefaultLocateOptions()
	options.Filters.IDs = in.Snapshots

	matches, err := locate.LocateSnapshotIDs(repo, options)
	if err != nil {
		return removeSnapshotsOutput{}, err
	}
	if len(matches) == 0 {
		return removeSnapshotsOutput{}, fmt.Errorf("no snapshot matched the given IDs")
	}

	out := removeSnapshotsOutput{
		Snapshots: describeSnapshots(repo, matches),
		Applied:   in.Apply,
	}

	if !in.Apply {
		return out, nil
	}

	if err := deleteSnapshots(ctx, repo, matches); err != nil {
		return removeSnapshotsOutput{}, err
	}

	return out, nil
}

func pruneSnapshots(ctx *appcontext.AppContext, repo *repository.Repository, in pruneSnapshotsInput) (pruneSnapshotsOutput, error) {
	options := locate.NewDefaultLocateOptions()

	_, reasons, err := locate.Match(repo, options)
	if err != nil {
		return pruneSnapshotsOutput{}, err
	}

	toDelete := make([]objects.MAC, 0, len(reasons))
	for id, reason := range reasons {
		if reason.Action == "delete" {
			toDelete = append(toDelete, id)
		}
	}

	out := pruneSnapshotsOutput{
		Snapshots: describeSnapshots(repo, toDelete),
		Applied:   in.Apply,
	}

	if !in.Apply || len(toDelete) == 0 {
		out.Applied = in.Apply && len(toDelete) > 0
		return out, nil
	}

	if err := deleteSnapshots(ctx, repo, toDelete); err != nil {
		return pruneSnapshotsOutput{}, err
	}

	return out, nil
}

// describeSnapshots renders the snapshots a destructive call is about to touch,
// so the caller can see what it asked for before it says apply.
func describeSnapshots(repo *repository.Repository, ids []objects.MAC) []removedSnapshot {
	described := make([]removedSnapshot, 0, len(ids))
	for _, id := range ids {
		entry := removedSnapshot{ID: hex.EncodeToString(id[:4])}

		snap, err := snapshot.Load(repo, id)
		if err == nil {
			entry.Timestamp = snap.Header.Timestamp.UTC().Format(time.RFC3339)
			entry.Source = snap.Header.GetSource(0).Importer.Directory
			snap.Close()
		}

		described = append(described, entry)
	}
	return described
}

func deleteSnapshots(ctx *appcontext.AppContext, repo *repository.Repository, ids []objects.MAC) error {
	var (
		mtx      sync.Mutex
		wg       sync.WaitGroup
		failures int
	)

	repo.NoStateToLocalDisk = true

	for _, id := range ids {
		wg.Add(1)
		go func(snapshotID objects.MAC) {
			defer wg.Done()
			if err := repo.DeleteSnapshot(snapshotID); err != nil {
				ctx.GetLogger().Error("%s", err)
				mtx.Lock()
				failures++
				mtx.Unlock()
				return
			}
			ctx.GetLogger().Info("mcp: removal of %x completed successfully", snapshotID[:4])
		}(id)
	}
	wg.Wait()

	if failures != 0 {
		return fmt.Errorf("failed to remove %d snapshots", failures)
	}

	return nil
}
