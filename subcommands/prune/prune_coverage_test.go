package prune

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// Parse with neither a filter nor a retention option is rejected.
func TestPruneParseNoFilter(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap1, snap2, ctx := generateRepoAndTwoSnaps(t, bufOut, bufErr)
	defer snap1.Close()
	defer snap2.Close()
	_ = repo

	cmd := &Prune{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no filter specified")
}

// Referencing a policy that does not exist surfaces a "policy not found" error.
func TestPruneParseUnknownPolicy(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap1, snap2, ctx := generateRepoAndTwoSnaps(t, bufOut, bufErr)
	defer snap1.Close()
	defer snap2.Close()
	_ = repo

	// ConfigDir defaults to a path with no policies.yml, so loading fails.
	ctx.ConfigDir = t.TempDir()

	cmd := &Prune{}
	err := cmd.Parse(ctx, []string{"-policy", "does-not-exist"})
	require.Error(t, err)
}

// A dry-run with a retention generous enough to keep everything prints a plan
// with zero deletions.
func TestPruneDryRunKeepsAll(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap1, snap2, ctx := generateRepoAndTwoSnaps(t, bufOut, bufErr)
	defer snap1.Close()
	defer snap2.Close()

	// Keep up to 10 per minute → both snapshots are kept.
	cmd := &Prune{}
	require.NoError(t, cmd.Parse(ctx, []string{"--per-minute=10"}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	out := bufOut.String()
	require.Contains(t, out, "prune: would keep 2 and delete 0 snapshot(s)")
	require.NotContains(t, out, "prune: removal of")
}

// -apply with a retention that deletes nothing returns early with status 0 and
// no removal output.
func TestPruneApplyNoDeletions(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap1, snap2, ctx := generateRepoAndTwoSnaps(t, bufOut, bufErr)
	defer snap1.Close()
	defer snap2.Close()

	cmd := &Prune{}
	require.NoError(t, cmd.Parse(ctx, []string{"-apply", "--per-minute=10"}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NotContains(t, bufOut.String(), "prune: removal of")
}
