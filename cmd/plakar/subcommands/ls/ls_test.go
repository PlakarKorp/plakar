package ls

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot"
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
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	return repo, snap
}

func TestExecuteCmdLsDefault(t *testing.T) {
	// Create a pipe to capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	repo, snap := generateSnapshot(t)
	defer snap.Close()

	args := []string{}

	subcommand := &Ls{}
	err = subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "ls", subcommand.Name())

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Close the write end of the pipe and restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))
	fields := strings.Fields(lines[0])
	require.Equal(t, 6, len(fields))
	require.Equal(t, snap.Header.Timestamp.Local().Format(time.RFC3339), fields[0])
	require.Equal(t, hex.EncodeToString(snap.Header.GetIndexShortID()), fields[1])
	require.Equal(t, snap.Header.GetSource(0).Importer.Directory, fields[len(fields)-1])
}

func TestExecuteCmdLsFilterByIDAndRecursive(t *testing.T) {
	// Create a pipe to capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	repo, snap := generateSnapshot(t)
	defer snap.Close()

	args := []string{"-recursive", hex.EncodeToString(snap.Header.GetIndexShortID())}

	subcommand := &Ls{}
	err = subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Close the write end of the pipe and restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 2, len(lines))
	// last line should have the filename we backed up
	lastline := lines[len(lines)-1]
	fields := strings.Fields(lastline)
	require.Equal(t, 7, len(fields))
	// disable timestamp testing because it can make the test flaky if the test ran in the last second
	// require.Equal(t, snap.Header.Timestamp.Local().Format(time.RFC3339), fields[0])
	require.Equal(t, fmt.Sprintf("%s/subdir/dummy.txt", snap.Header.GetSource(0).Importer.Directory), fields[len(fields)-1])
}

func TestExecuteCmdLsFilterUuid(t *testing.T) {
	// Create a pipe to capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	repo, snap := generateSnapshot(t)
	defer snap.Close()

	args := []string{"-uuid"}

	subcommand := &Ls{}
	err = subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Close the write end of the pipe and restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))
	fields := strings.Fields(lines[0])
	require.Equal(t, 6, len(fields))
	require.Equal(t, snap.Header.Timestamp.Local().Format(time.RFC3339), fields[0])
	indexId := snap.Header.GetIndexID()
	require.Equal(t, hex.EncodeToString(indexId[:]), fields[1])
	require.Equal(t, snap.Header.GetSource(0).Importer.Directory, fields[len(fields)-1])
}
