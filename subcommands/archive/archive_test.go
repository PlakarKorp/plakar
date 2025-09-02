package archive

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
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
	args := []string{"-output", outputDir, fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}

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
