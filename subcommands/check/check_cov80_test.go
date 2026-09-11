package check

import (
	"bytes"
	"encoding/hex"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

// Execute with no positional snapshots locates every snapshot in the repo via
// LocateSnapshotIDs and checks each one. Two snapshots exercise the
// multi-iteration loop in the len(cmd.Snapshots)==0 branch.
func TestCheckCov80ExecuteAllSnapshots(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Add a second snapshot so the all-snapshots loop runs twice.
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("extra.txt", 0644, "extra content"),
	})
	defer snap2.Close()

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)

	cmd := &Check{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Empty(t, cmd.Snapshots)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// Execute with an explicit, valid snapshot ID and a sub-path runs the
// positional-args branch with ParseSnapshotPath returning a non-empty path.
func TestCheckCov80ExecuteExplicitIDWithPath(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)

	indexID := snap.Header.GetIndexID()
	cmd := &Check{}
	require.NoError(t, cmd.Parse(ctx, []string{hex.EncodeToString(indexID[:]) + ":/another_subdir"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// Both -fast and -no-verify combined: fast check with verification disabled.
func TestCheckCov80ExecuteFastNoVerify(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)

	cmd := &Check{}
	require.NoError(t, cmd.Parse(ctx, []string{"-fast", "-no-verify"}))
	require.True(t, cmd.FastCheck)
	require.True(t, cmd.NoVerify)
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
