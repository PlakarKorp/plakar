package locate

import (
	"bytes"
	"encoding/hex"
	"os"
	"strings"
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

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot) {
	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
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

func TestExecuteCmdLocateDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{"dummy.txt"}

	subcommand := &Locate{}
	err := subcommand.Parse(repo.AppContext(), args)

	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "locate", subcommand.Name())

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.NotNil(t, status)

	// output should look like this
	// d92a4c73:/tmp/tmp_to_backup1424943315/subdir/dummy.txt

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))
}

func TestExecuteCmdLocateWithSnapshotId(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{"-snapshot", hex.EncodeToString(snap.Header.GetIndexShortID()), "dummy.txt"}

	subcommand := &Locate{}
	err := subcommand.Parse(repo.AppContext(), args)

	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.NotNil(t, status)

	// output should look like this
	// d92a4c73:/tmp/tmp_to_backup1424943315/subdir/dummy.txt

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))
}
