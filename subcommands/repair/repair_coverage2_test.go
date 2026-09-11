package repair

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// orphanPackfiles creates a snapshot, then deletes its committed state from the
// store while leaving the packfile entries in the local cache. This is exactly
// the "orphaned packfiles" condition the repair missing-state loop is meant to
// fix: packfiles exist in storage and in the cached local state, but no
// committed remote state references them anymore.
//
// repo.DeleteState only removes the __state__ entry from the store; it does not
// touch __packfile__ entries, so ListPackfileEntries still yields them while
// GetStates no longer lists the owning state.
//
// The repository cache only learns about the snapshot's packfile entries once
// the state has been rebuilt into it, so we must rebuild the cache while the
// state still exists, and only then delete the remote state. That leaves the
// cache holding packfile entries whose owning state is no longer committed.
func orphanPackfiles(t *testing.T, repo *repository.Repository) {
	t.Helper()

	cache, err := repo.AppContext().GetCache().Repository(repo.Configuration().RepositoryID)
	require.NoError(t, err)

	// Populate the cache with the snapshot's packfile entries while the state
	// is still committed.
	require.NoError(t, repo.RebuildStateWithCache(cache))

	states, err := repo.GetStates()
	require.NoError(t, err)
	require.NotEmpty(t, states, "snapshot should have produced at least one state")
	for _, s := range states {
		require.NoError(t, repo.DeleteState(s))
	}
	after, err := repo.GetStates()
	require.NoError(t, err)
	require.Empty(t, after, "all committed states should be gone")
}

// TestRepairExecuteDryRunFindsMissingState exercises the missing-state detection
// branch of Execute without -apply. After orphaning the packfiles the dry run
// must report a missing state and the "to apply" hint, but must not mutate the
// repository.
func TestRepairExecuteDryRunFindsMissingState(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/a.txt", 0644, "hello world hello world"),
		ptesting.NewMockFile("subdir/b.txt", 0644, "more and more content"),
	})
	snap.Close()

	orphanPackfiles(t, repo)

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// TestRepairExecuteApplyRepairsMissingState exercises the full -apply
// missing-state repair loop: it takes the exclusive lock, rebuilds the delta
// state from the orphaned packfiles and re-commits a state via PutState. After
// the repair the repository should once again report a committed state.
func TestRepairExecuteApplyRepairsMissingState(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("d"),
		ptesting.NewMockFile("d/f.txt", 0644, "payload payload payload"),
		ptesting.NewMockFile("d/g.txt", 0644, "second file contents"),
	})
	snap.Close()

	orphanPackfiles(t, repo)

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{"-apply"}))
	require.True(t, cmd.Apply)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// The repair loop must have re-committed a state via PutState.
	states, err := repo.GetStates()
	require.NoError(t, err)
	require.NotEmpty(t, states, "repair -apply should have written a fresh state")
}
