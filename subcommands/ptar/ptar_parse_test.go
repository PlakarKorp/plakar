package ptar

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/integration-ptar/storage"
	"github.com/PlakarKorp/plakar/config"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestParseMissingOutput(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-plaintext", "/some/path"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "-o option must be specified")
}

func TestParseUnknownHashingAlgo(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-plaintext", "-hashing", "not-a-real-algo",
		"-o", filepath.Join(tmpDir, "x.ptar"), "/some/path"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown hashing algorithm")
}

func TestParsePassphraseFromEnv(t *testing.T) {
	t.Setenv("PLAKAR_PASSPHRASE", "correct horse battery staple")
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-o", filepath.Join(tmpDir, "x.ptar"), "/some/path"})
	require.NoError(t, err)
	require.Equal(t, []byte("correct horse battery staple"), cmd.RepositorySecret)
}

func TestParseEmptyPassphraseFromEnv(t *testing.T) {
	t.Setenv("PLAKAR_PASSPHRASE", "")
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-o", filepath.Join(tmpDir, "x.ptar"), "/some/path"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty passphrase")
}

func TestParseDefaultsToCWD(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.CWD = "/the/working/dir"
	tmpDir := t.TempDir()
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-plaintext", "-o", filepath.Join(tmpDir, "x.ptar")})
	require.NoError(t, err)
	require.Equal(t, []string{"/the/working/dir"}, []string(cmd.BackupTargets))
}

func TestParseMultipleTargets(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-plaintext", "-o", filepath.Join(tmpDir, "x.ptar"),
		"/path/one", "/path/two"})
	require.NoError(t, err)
	require.Equal(t, []string{"/path/one", "/path/two"}, []string(cmd.BackupTargets))
}

func TestExecuteRefusesOverwrite(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpSourceDir := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})

	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "test.ptar")

	args := []string{"-plaintext", "-o", out, filepath.Join(tmpSourceDir, "subdir")}
	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, args))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// A second run with the same output must refuse without -overwrite.
	cmd2 := &Ptar{}
	require.NoError(t, cmd2.Parse(ctx, args))
	status, err = cmd2.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "already exists")
}

func TestExecuteOverwrite(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpSourceDir := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})

	tmpDir := t.TempDir()
	out := filepath.Join(tmpDir, "test.ptar")

	require.NoError(t, os.WriteFile(out, []byte("stale archive"), 0644))

	args := []string{"-plaintext", "-overwrite", "-o", out, filepath.Join(tmpSourceDir, "subdir")}
	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, args))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestExecuteNoCompression(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpSourceDir := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})

	tmpDir := t.TempDir()
	args := []string{"-plaintext", "-no-compression", "-o",
		filepath.Join(tmpDir, "test.ptar"), filepath.Join(tmpSourceDir, "subdir")}
	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, args))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestParseSyncTargetUnknownRepository(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.Config = config.NewConfig()
	tmpDir := t.TempDir()
	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-plaintext", "-o", filepath.Join(tmpDir, "x.ptar"),
		"-k", "@does-not-exist"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "peer repository")
}

func TestExecuteUnresolvableSource(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	ctx.Config = config.NewConfig()

	tmpDir := t.TempDir()
	args := []string{"-plaintext", "-o", filepath.Join(tmpDir, "test.ptar"), "@nonexistent"}
	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, args))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "could not resolve importer")
}
