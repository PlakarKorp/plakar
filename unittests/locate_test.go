package unittests

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/locate"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot) {
	repo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	return repo, snap
}

func generateSnapshotWithMetadata(t *testing.T, repo *repository.Repository, opts ptesting.TestingOptions) *snapshot.Snapshot {
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	}, opts)
	require.NotNil(t, snap)
	return snap
}

func TestParseSnapshotPath(t *testing.T) {
	// Test case: Snapshot path with prefix and pattern
	prefix, pattern := locate.ParseSnapshotPath("prefix:pattern")
	require.Equal(t, "prefix", prefix)
	require.Equal(t, "pattern", pattern)

	// Test case: Absolute path
	prefix, pattern = locate.ParseSnapshotPath("/absolute/path")
	require.Equal(t, "", prefix)
	require.Equal(t, "/absolute/path", pattern)

	// Test case: Only prefix without pattern
	prefix, pattern = locate.ParseSnapshotPath("prefix")
	require.Equal(t, "prefix", prefix)
	require.Equal(t, "", pattern)

	// Test case: Empty input
	prefix, pattern = locate.ParseSnapshotPath("")
	require.Equal(t, "", prefix)
	require.Equal(t, "", pattern)
}

func TestLookupSnapshotByPrefix(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Test case: Prefix that matches a single snapshot
	results := locate.LookupSnapshotByPrefix(repo, hex.EncodeToString(snap.Header.GetIndexShortID()))
	require.Len(t, results, 1)
	require.Equal(t, results[0], snap.Header.Identifier)

	// Test case: Prefix that matches no snapshots
	results = locate.LookupSnapshotByPrefix(repo, hex.EncodeToString([]byte{snap.Header.Identifier[0] + 1}))
	require.Len(t, results, 0)
}

func TestLocateSnapshotByPrefix(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	found, err := locate.LocateSnapshotByPrefix(repo, hex.EncodeToString(snap.Header.GetIndexShortID()))
	require.NoError(t, err)
	require.Equal(t, found, snap.Header.Identifier)

	_, err = locate.LocateSnapshotByPrefix(repo, hex.EncodeToString([]byte{snap.Header.Identifier[0] + 1}))
	require.ErrorContains(t, err, "no snapshot has prefix")
}

func TestOpenSnapshotByPath(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	prefix := hex.EncodeToString(snap.Header.GetIndexShortID())
	snapshotPath := fmt.Sprintf("%s:/subdir/dummy.txt", prefix)
	snap, snapRoot, err := locate.OpenSnapshotByPath(repo, snapshotPath)
	require.NoError(t, err)
	require.NotNil(t, snap)
	require.Equal(t, filepath.Clean("/subdir/dummy.txt"), snapRoot)

	// Test case: Invalid prefix
	snapshotPath = "invalid:/subdir/dummy.txt"
	_, _, err = locate.OpenSnapshotByPath(repo, snapshotPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no snapshot has prefix")
}

func TestLocateSnapshotIDs(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	// Create a temporary repository
	repo, _ := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	// Generate mock snapshots with metadata
	snap1 := generateSnapshotWithMetadata(t, repo, ptesting.WithName("snapshot1"))
	defer snap1.Close()

	snap2 := generateSnapshotWithMetadata(t, repo, ptesting.WithName("snapshot2"))
	defer snap2.Close()

	snap3 := generateSnapshotWithMetadata(t, repo, ptesting.WithName("snapshot3"))
	defer snap3.Close()

	// Test case: Locate snapshots by category
	opts := &locate.LocateOptions{
		MaxConcurrency: 1,
		Name:           "snapshot2",
	}
	results, err := locate.LocateSnapshotIDs(repo, opts)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Contains(t, results, snap2.Header.Identifier)

	// Test case: Locate latest snapshot
	opts = &locate.LocateOptions{
		MaxConcurrency: 1,
		Latest:         true,
		SortOrder:      locate.LocateSortOrderDescending,
	}
	results2, err := locate.LocateSnapshotIDs(repo, opts)
	require.NoError(t, err)
	require.Len(t, results2, 1)
	require.Contains(t, results2, snap3.Header.Identifier)
}
