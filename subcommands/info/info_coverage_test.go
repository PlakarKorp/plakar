package info

import (
	"bytes"
	"encoding/hex"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// executeRepository over an encrypted repo exercises the Encryption branch
// (SubkeyAlgorithm, Canary, KDF/ARGON2ID params) that the unencrypted default
// test never reaches.
func TestExecuteCmdInfoEncryptedRepository(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	passphrase := []byte("correct-horse-battery-staple")
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, &passphrase)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	defer snap.Close()

	cmd := &Info{}
	require.NoError(t, cmd.Parse(ctx, []string{}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "Encryption:")
	require.Contains(t, out, "KDF: ARGON2ID")
	require.Contains(t, out, "Snapshots: 1")
}

// An unknown snapshot id must surface as an error from executeSnapshot.
func TestExecuteCmdInfoSnapshotNotFound(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Info{SnapshotID: "deadbeefdeadbeefdeadbeefdeadbeef"}
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// info -errors on an unknown snapshot id goes through executeErrors and errors
// out on snapshot open.
func TestExecuteCmdInfoErrorsSnapshotNotFound(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Info{SnapshotID: "deadbeefdeadbeefdeadbeefdeadbeef", Errors: true}
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// info -errors on a valid snapshot with no errors walks executeErrors to
// completion (returns 0 and prints nothing about errors).
func TestExecuteCmdInfoErrorsValidSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	cmd := &Info{Errors: true}
	require.NoError(t, cmd.Parse(ctx, []string{"-errors"}))
	cmd.SnapshotID = hex.EncodeToString(indexID[:])

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// Parse rejects more than one positional argument.
func TestExecuteCmdInfoTooManyArgs(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()
	_ = repo

	cmd := &Info{}
	err := cmd.Parse(ctx, []string{"a", "b"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}
