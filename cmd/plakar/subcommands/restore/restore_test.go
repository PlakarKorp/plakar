package restore

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T) (*repository.Repository, *snapshot.Snapshot) {
	repo := ptesting.GenerateRepository(t, nil, nil, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	return repo, snap
}

func checkRestored(t *testing.T, restoreDir string) {
	files := map[string]string{
		filepath.FromSlash("subdir/dummy.txt"):       "hello dummy",
		filepath.FromSlash("subdir/foo.txt"):         "hello foo",
		filepath.FromSlash("another_subdir/bar.txt"): "hello bar",
	}
	dirs := []string{"subdir", "another_subdir"}

	for file, exp := range files {
		dest := filepath.Join(restoreDir, file)

		content, err := os.ReadFile(dest)
		require.NoError(t, err)
		require.Equal(t, exp, string(content))

		require.NoError(t, os.Remove(dest))
	}

	for _, dir := range dirs {
		dest := filepath.Join(restoreDir, dir)
		require.NoError(t, os.Remove(dest), "directory not empty?")
	}

	rest, err := os.ReadDir(restoreDir)
	require.NoError(t, err)
	require.Empty(t, rest)
}

func TestExecuteCmdRestoreDefault(t *testing.T) {
	repo, snap := generateSnapshot(t)
	defer snap.Close()

	tmpToRestoreDir, err := os.MkdirTemp("", "tmp_to_restore")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpToRestoreDir)
	})

	args := []string{"-to", tmpToRestoreDir}

	subcommand, err := parse_cmd_restore(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "restore", subcommand.(*Restore).Name())

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	checkRestored(t, tmpToRestoreDir)
}

func TestExecuteCmdRestoreSpecificSnapshot(t *testing.T) {
	// create one snapshot
	repo, snap := generateSnapshot(t)
	defer snap.Close()

	tmpToRestoreDir, err := os.MkdirTemp("", "tmp_to_restore")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpToRestoreDir)
	})

	indexId := snap.Header.GetIndexID()
	args := []string{"-to", tmpToRestoreDir, fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}
	subcommand, err := parse_cmd_restore(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "restore", subcommand.(*Restore).Name())

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	checkRestored(t, tmpToRestoreDir)
}
