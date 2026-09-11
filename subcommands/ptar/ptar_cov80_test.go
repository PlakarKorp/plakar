package ptar

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/importer"
	_ "github.com/PlakarKorp/integrations/ptar/storage"
	"github.com/PlakarKorp/plakar/config"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// TestCov80PtarParseUnknownPeerInConfig drives the GetRepository failure branch
// of the -k loop in Parse: the referenced peer is not configured and is not a
// resolvable bare location, so Parse returns a "peer repository" error before
// any backup happens.
func TestCov80PtarParseUnknownPeerInConfig(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.Config = config.NewConfig()
	tmpDir := t.TempDir()

	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{
		"-plaintext",
		"-o", filepath.Join(tmpDir, "a.ptar"),
		"-k", "@no-such-peer",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "peer repository")
}

// TestCov80PtarParseEncryptedPeerConfigPassphrase drives the encrypted-peer
// branch of the -k loop: the peer kloset is encrypted and its passphrase is
// supplied through the config. Parse must derive the key, pass the canary
// check, and record the derived secret in cmd.SyncSecrets.
func TestCov80PtarParseEncryptedPeerConfigPassphrase(t *testing.T) {
	peerPass := []byte("peer-secret-pass")
	srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, &peerPass)

	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.Config = config.NewConfig()
	ctx.Config.Repositories["peer"] = map[string]string{
		"location":   srcRepo.Root(),
		"passphrase": string(peerPass),
	}
	tmpDir := t.TempDir()

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-plaintext",
		"-o", filepath.Join(tmpDir, "a.ptar"),
		"-k", "@peer",
	}))
	require.Len(t, cmd.SyncSecrets, 1)
	require.NotEmpty(t, cmd.SyncSecrets[0])
}

// TestCov80PtarParseEncryptedPeerWrongPassphrase drives the VerifyCanary failure
// branch of the encrypted-peer path: the config supplies the wrong passphrase
// so Parse must reject with "invalid passphrase".
func TestCov80PtarParseEncryptedPeerWrongPassphrase(t *testing.T) {
	peerPass := []byte("peer-secret-pass")
	srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, &peerPass)

	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.Config = config.NewConfig()
	ctx.Config.Repositories["peer"] = map[string]string{
		"location":   srcRepo.Root(),
		"passphrase": "wrong-passphrase",
	}
	tmpDir := t.TempDir()

	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{
		"-plaintext",
		"-o", filepath.Join(tmpDir, "a.ptar"),
		"-k", "@peer",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid passphrase")
}

// TestCov80PtarSyncEncryptedPeerEndToEnd runs a full ptar archive build that
// pulls a snapshot from an encrypted peer kloset (-k) configured with its
// passphrase. This drives both Parse and Execute's encrypted synchronize path
// (cmd.SyncSecrets is fed into repository.New for the source).
func TestCov80PtarSyncEncryptedPeerEndToEnd(t *testing.T) {
	peerPass := []byte("peer-secret-pass")
	srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, &peerPass)
	srcSnap := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	defer srcSnap.Close()

	dstRepo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.Config = config.NewConfig()
	ctx.Config.Repositories["peer"] = map[string]string{
		"location":   srcRepo.Root(),
		"passphrase": string(peerPass),
	}
	tmpDir := t.TempDir()

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-plaintext",
		"-o", filepath.Join(tmpDir, "enc-sync.ptar"),
		"-k", "@peer",
	}))
	status, err := cmd.Execute(ctx, dstRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// The archive must exist on disk.
	_, statErr := os.Stat(filepath.Join(tmpDir, "enc-sync.ptar"))
	require.NoError(t, statErr)
}

// TestCov80PtarExecuteUnknownHashingConfig drives the Execute hashing-lookup
// failure branch: the command struct is built with an unknown hashing algorithm
// (bypassing the Parse-time guard) so LookupDefaultConfiguration fails.
func TestCov80PtarExecuteUnknownHashing(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()

	cmd := &Ptar{
		KlosetPath:   filepath.Join(tmpDir, "bad.ptar"),
		NoEncryption: true,
		Hashing:      "NOSUCHALGO",
	}
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}
