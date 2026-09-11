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

// ---------- Parse ----------

func TestPtarParseMissingOutput(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-plaintext", "/some/path"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "-o option must be specified")
}

func TestPtarParseUnknownHashing(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{
		"-plaintext",
		"-hashing", "NOSUCHALGO",
		"-o", filepath.Join(tmpDir, "a.ptar"),
		tmpDir,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown hashing algorithm")
}

func TestPtarParsePassphraseFromEnvNonEmpty(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("PLAKAR_PASSPHRASE", "correct horse battery staple")

	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-o", filepath.Join(tmpDir, "a.ptar"), tmpDir})
	require.NoError(t, err)
	require.Equal(t, []byte("correct horse battery staple"), cmd.GetRepositorySecret())
}

func TestPtarParsePassphraseFromEnvEmpty(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	t.Setenv("PLAKAR_PASSPHRASE", "")

	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-o", filepath.Join(tmpDir, "a.ptar"), tmpDir})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty passphrase")
}

func TestPtarParseDefaultsToCWD(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	ctx.CWD = "/the/current/dir"

	cmd := &Ptar{}
	// -plaintext avoids the passphrase prompt; no targets -> defaults to CWD.
	err := cmd.Parse(ctx, []string{"-plaintext", "-o", filepath.Join(tmpDir, "a.ptar")})
	require.NoError(t, err)
	require.Equal(t, []string{"/the/current/dir"}, []string(cmd.BackupTargets))
}

func TestPtarParseMultipleTargets(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()

	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{
		"-plaintext",
		"-o", filepath.Join(tmpDir, "a.ptar"),
		"/path/one", "/path/two", "/path/three",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"/path/one", "/path/two", "/path/three"}, []string(cmd.BackupTargets))
}

func TestPtarParseUnknownPeer(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.Config = config.NewConfig()
	tmpDir := t.TempDir()

	cmd := &Ptar{}
	// -k references a peer that is not configured; opening it as a bare
	// location fails before any backup happens.
	err := cmd.Parse(ctx, []string{
		"-plaintext",
		"-o", filepath.Join(tmpDir, "a.ptar"),
		"-k", "does-not-exist",
	})
	require.Error(t, err)
}

// ---------- Execute ----------

func TestPtarExecuteRefuseOverwrite(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	src := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})

	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "exists.ptar")
	require.NoError(t, os.WriteFile(out, []byte("preexisting"), 0644))

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{"-plaintext", "-o", out, filepath.Join(src, "subdir")}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "already exists")
}

func TestPtarExecuteOverwrite(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	src := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})

	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "exists.ptar")
	require.NoError(t, os.WriteFile(out, []byte("preexisting"), 0644))

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{"-plaintext", "-overwrite", "-o", out, filepath.Join(src, "subdir")}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestPtarExecuteNoCompression(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	src := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})

	tmpDir := t.TempDir()
	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-plaintext", "-no-compression",
		"-o", filepath.Join(tmpDir, "nc.ptar"),
		filepath.Join(src, "subdir"),
	}))
	require.True(t, cmd.NoCompression)
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestPtarExecuteUnresolvableSource(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()

	// GenerateRepositoryWithoutConfig leaves ctx.Config nil; the @source
	// resolution path dereferences it, so install an empty config.
	ctx.Config = config.NewConfig()

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{"-plaintext", "-o", filepath.Join(tmpDir, "u.ptar")}))
	// Force a backup target that uses the @source syntax pointing at an
	// undefined source; backup() must fail to resolve the importer.
	cmd.BackupTargets = listFlag{"@nonexistent-source"}
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "could not resolve importer")
}
