package cat

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdCatDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap.Close()

	args := []string{":subdir/dummy.txt"}

	subcommand := &Cat{}
	err := subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	if err != nil {
		t.Fatal("got an error: ", err)
	}
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Equal(t, "hello dummy", output)
}

func TestExecuteCmdCatErrorAmbiguous(t *testing.T) {
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
	snap = ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap.Close()

	args := []string{":subdir/dummy.txt"}

	subcommand := &Cat{}
	err := subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.Error(t, err, "errors occurred")
	require.Equal(t, 1, status)

	outputErr := bufErr.String()
	require.Contains(t, outputErr, "snapshot ID is ambiguous:  (matches 2 snapshots)")
}

func TestExecuteCmdCatErrorNotRegularFile(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap.Close()

	args := []string{fmt.Sprintf("%s:/", hex.EncodeToString(snap.Header.GetIndexShortID()))}

	subcommand := &Cat{}
	err := subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.Error(t, err, "errors occurred")
	require.Equal(t, 1, status)

	outputErr := bufErr.String()
	require.Contains(t, outputErr, "not a regular file")
}

func TestExecuteCmdCatErrorUnknownFile(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap.Close()

	args := []string{fmt.Sprintf("%s:/unknown", hex.EncodeToString(snap.Header.GetIndexShortID()))}

	subcommand := &Cat{}
	err := subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.Error(t, err, "errors occurred")
	require.Equal(t, 1, status)

	outputErr := bufErr.String()
	require.Contains(t, outputErr, "cat: /unknown: no such file")
}

func TestExecuteCmdCatHighlight(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar", 0644, "hello bar"),
	})
	snap.Close()

	args := []string{"--highlight", ":subdir/dummy.txt"}

	subcommand := &Cat{}
	err := subcommand.Parse(repo.AppContext(), args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(repo.AppContext(), repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Equal(t, "\x1b[1m\x1b[37mhello dummy\x1b[0m", output)
}
