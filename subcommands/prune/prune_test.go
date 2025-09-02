package prune

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateRepoAndTwoSnaps(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *snapshot.Snapshot, *appcontext.AppContext) {
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	// First snapshot
	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/a.txt", 0644, "hello A"),
	})

	// Second snapshot (newest)
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/b.txt", 0644, "hello B"),
	})

	return repo, snap1, snap2, ctx
}

func TestPrune_DryRun_PerMinuteCap(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap1, snap2, ctx := generateRepoAndTwoSnaps(t, bufOut, bufErr)
	defer snap1.Close()
	defer snap2.Close()

	// Cap 1 per minute across all minute buckets. With two snaps in the same minute,
	// prune will keep the newest and mark the older for delete — but dry-run prints a plan only.
	args := []string{"--per-minute=1"}

	cmd := &Prune{}
	err := cmd.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()

	// In dry-run, we get a summary like:
	//   prune: would keep X and delete Y snapshot(s), run with -apply to proceed
	require.Contains(t, out, "prune: would keep 1 and delete 1 snapshot(s)")
	// Should list minute matches/caps
	require.Contains(t, out, "match=minute:")
	require.Contains(t, out, "cap=1")
	// Should NOT have the actual removal line without -apply
	require.NotContains(t, out, "prune: removal of")
}

func TestPrune_Apply_PerMinuteCap(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap1, snap2, ctx := generateRepoAndTwoSnaps(t, bufOut, bufErr)
	defer snap1.Close()
	defer snap2.Close()

	// With -apply the older snapshot should actually be removed.
	// Retention keeps the newest in the minute; snap1 is older → deleted.
	args := []string{"-apply", "--per-minute=1"}

	cmd := &Prune{}
	err := cmd.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()

	// prune logs via the app context logger:
	//   info: rm: removal of <first 4 bytes hex> completed successfully
	// The "short id" from GetIndexShortID() should match those 4 bytes.
	short1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	require.Contains(t, out, fmt.Sprintf("info: prune: removal of %s completed successfully", short1))

	// Sanity: ensure it didn't claim to remove the newest one (kept)
	short2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	require.NotContains(t, out, fmt.Sprintf("info: prune: removal of %s completed successfully", short2))
}
