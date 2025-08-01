package ptar

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/plakar/connectors/ptar/storage"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdPtarDefault(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpSourceDir := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})

	tmpDir, err := os.MkdirTemp("", "tmp_ptar")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	args := []string{"-plaintext", "-o", filepath.Join(tmpDir, "test.ptar"), filepath.Join(tmpSourceDir, "subdir")}

	subcommand := &Ptar{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestExecuteCmdPtarWithSync(t *testing.T) {
	// Create source repository
	srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, nil)
	srcSnap := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	defer srcSnap.Close()

	// Create destination repository
	dstRepo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)

	tmpDir, err := os.MkdirTemp("", "tmp_ptar")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	location, err := srcRepo.Location()
	require.NoError(t, err)
	args := []string{"-plaintext", "-o", filepath.Join(tmpDir, "test.ptar"), "-k", location}

	subcommand := &Ptar{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, dstRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
