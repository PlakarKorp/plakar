package info

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

// TestExecuteCmdInfoEncryptedRepoCoverage4 drives `info` (no args ->
// executeRepository) against an ENCRYPTED repository so the
// `Encryption != nil` branch and the ARGON2ID KDF switch case are
// exercised (these are skipped by the default unencrypted helper repo).
func TestExecuteCmdInfoEncryptedRepoCoverage4(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	passphrase := []byte("correct horse battery staple")
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, &passphrase)

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	defer snap.Close()

	// sanity: this repo really is encrypted
	require.NotNil(t, repo.Configuration().Encryption)

	cmd := &Info{}
	require.NoError(t, cmd.Parse(ctx, []string{}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	// Lock in the encryption-block statements.
	require.Contains(t, output, "Encryption:")
	require.Contains(t, output, "SubkeyAlgorithm:")
	require.Contains(t, output, "DataAlgorithm:")
	require.Contains(t, output, "KDF: ARGON2ID")
	require.Contains(t, output, "Compression:")
	require.Contains(t, output, "Snapshots: 1")
}

// TestExecuteCmdInfoPlainRepoCoverage4 drives `info` against an
// unencrypted repo and asserts that NO encryption block is printed,
// locking in the `Encryption == nil` false-branch.
func TestExecuteCmdInfoPlainRepoCoverage4(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "hello"),
	})
	defer snap.Close()

	require.Nil(t, repo.Configuration().Encryption)

	cmd := &Info{}
	require.NoError(t, cmd.Parse(ctx, []string{}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.NotContains(t, output, "Encryption:")
	require.Contains(t, output, "Snapshots: 1")
	require.Contains(t, output, "Storage size:")
	require.Contains(t, output, "Logical size:")
}

// TestExecuteCmdInfoErrorsBadIDCoverage4 exercises the error return of
// executeErrors when OpenSnapshotByPath fails on a non-existent ID.
func TestExecuteCmdInfoErrorsBadIDCoverage4(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Info{}
	// 64 hex chars that do not match any snapshot.
	require.NoError(t, cmd.Parse(ctx, []string{"-errors",
		"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}))
	require.True(t, cmd.Errors)

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestExecuteCmdInfoSnapshotBadIDCoverage4 exercises the error return of
// executeSnapshot when OpenSnapshotByPath fails.
func TestExecuteCmdInfoSnapshotBadIDCoverage4(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Info{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}))
	require.False(t, cmd.Errors)
	require.NotEmpty(t, cmd.SnapshotID)

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestExecuteCmdInfoErrorsScanCoverage4 exercises executeErrors on a real
// snapshot (the fs.Errors iteration path) and asserts a clean run.
func TestExecuteCmdInfoErrorsScanCoverage4(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexID := snap.Header.GetIndexID()
	cmd := &Info{}
	require.NoError(t, cmd.Parse(ctx, []string{"-errors", hex.EncodeToString(indexID[:])}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
