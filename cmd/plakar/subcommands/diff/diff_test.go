package diff

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdDiffIdentical(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	// create one snapshot
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap.Close()

	// create second snapshot
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap2.Close()

	indexId1 := snap.Header.GetIndexShortID()
	indexId2 := snap2.Header.GetIndexShortID()
	backupDir1 := snap.Header.GetSource(0).Importer.Directory
	backupDir2 := snap2.Header.GetSource(0).Importer.Directory
	snapPath1 := fmt.Sprintf("%s:%s/subdir/dummy.txt", hex.EncodeToString(indexId1[:]), backupDir1)
	snapPath2 := fmt.Sprintf("%s:%s/subdir/dummy.txt", hex.EncodeToString(indexId2[:]), backupDir2)
	args := []string{snapPath1, snapPath2}

	subcommand := &Diff{}
	err := subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "diff", subcommand.Name())

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	outputErr := bufErr.String()
	require.Contains(t, outputErr, fmt.Sprintf("%s and %s are identical\n", snapPath1, snapPath2))
}

func TestExecuteCmdDiffFiles(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	// create one snapshot
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap.Close()

	// create second different snapshot
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dumpy"), // <- dumpy!
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap2.Close()

	indexId1 := snap.Header.GetIndexShortID()
	indexId2 := snap2.Header.GetIndexShortID()
	snapPath1 := fmt.Sprintf("%s:subdir/dummy.txt", hex.EncodeToString(indexId1[:]))
	snapPath2 := fmt.Sprintf("%s:subdir/dummy.txt", hex.EncodeToString(indexId2[:]))
	args := []string{snapPath1, snapPath2}

	subcommand := &Diff{}
	err := subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, `
@@ -1 +1 @@
-hello dummy
+hello dumpy`)
}
