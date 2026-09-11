package repair

import (
	"bytes"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// TestRepairParseCapturesSecret verifies that Parse copies the secret set on the
// AppContext into the command's RepositorySecret field.
func TestRepairParseCapturesSecret(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	secret := []byte("hunter2-top-secret")
	ctx.SetSecret(secret)

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Equal(t, secret, cmd.GetRepositorySecret())
}

// TestRepairExecuteDryRunEmptyRepo drives a dry-run on a freshly created
// repository with no snapshots. No repairs are needed and the command succeeds.
func TestRepairExecuteDryRunEmptyRepo(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// TestRepairExecuteDryRunPopulatedHealthy drives a dry-run on a healthy
// repository that already contains a snapshot. All packfiles belong to known
// remote states, so no missing-state repairs are reported.
func TestRepairExecuteDryRunPopulatedHealthy(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/a.txt", 0644, "hello world"),
		ptesting.NewMockFile("subdir/b.txt", 0644, "more content here"),
	})
	defer snap.Close()

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// TestRepairExecuteApplyHealthy drives repair with -apply against a healthy
// repository. This exercises the exclusive Lock/Unlock path. Because the
// repository is healthy there are no missing states to repair, so the
// orphaned-packfile loop is skipped.
func TestRepairExecuteApplyHealthy(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("d"),
		ptesting.NewMockFile("d/f.txt", 0644, "payload"),
	})
	defer snap.Close()

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{"-apply"}))
	require.True(t, cmd.Apply)
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
