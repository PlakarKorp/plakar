package archive

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestArchiveParseNoSnapshot(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo
	cmd := &Archive{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one snapshot")
}

func TestArchiveParseUnsupportedFormat(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo
	cmd := &Archive{}
	err := cmd.Parse(ctx, []string{"-format", "rar", "abc"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported format")
}

func TestArchiveParseDefaultOutputName(t *testing.T) {
	// With no -output, Parse derives a plakar-<ts>.<ext> filename from the
	// format.
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo
	cmd := &Archive{}
	require.NoError(t, cmd.Parse(ctx, []string{"-format", "zip", "abc"}))
	require.Contains(t, cmd.Output, "plakar-")
	require.Contains(t, cmd.Output, ".zip")
}

func TestArchiveExecuteBadSnapshot(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "a"),
	})
	defer snap.Close()

	cmd := &Archive{}
	require.NoError(t, cmd.Parse(ctx, []string{"-output", filepath.Join(t.TempDir(), "x.tar.gz"), "deadbeefdeadbeef"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "could not open snapshot")
}

func TestArchiveExecuteOutputCreateError(t *testing.T) {
	// An output path under a nonexistent directory makes os.Create fail.
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "a"),
	})
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	cmd := &Archive{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-output", "/nonexistent-dir-xyz/sub/archive.tar.gz",
		hex.EncodeToString(indexId[:]),
	}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "failed to create")
}

func TestExecuteCmdArchiveDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	defer snap.Close()

	tmpDestinationDir, err := os.MkdirTemp("", "archive_destination")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDestinationDir)
	})

	indexId := snap.Header.GetIndexID()
	outputDir := fmt.Sprintf("%s/archive_test", tmpDestinationDir)
	args := []string{"-output", outputDir, hex.EncodeToString(indexId[:])}

	subcommand := &Archive{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	_, err = os.Stat(outputDir)
	require.NoError(t, err)
}
