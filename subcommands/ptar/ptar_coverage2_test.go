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

// ---------- listFlag ----------

func TestCov2ListFlagDedupeAndString(t *testing.T) {
	var l listFlag
	require.NoError(t, l.Set("a"))
	require.NoError(t, l.Set("b"))
	require.NoError(t, l.Set("a")) // duplicate is silently ignored
	require.Equal(t, []string{"a", "b"}, []string(l))
	require.Equal(t, "[a b]", l.String())
}

// ---------- Parse: passphrase from key file ----------

func TestCov2ParseKeyFromFile(t *testing.T) {
	_, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	tmpDir := t.TempDir()
	ctx.KeyFromFile = "a key from file"

	cmd := &Ptar{}
	err := cmd.Parse(ctx, []string{"-o", filepath.Join(tmpDir, "a.ptar"), tmpDir})
	require.NoError(t, err)
	require.Equal(t, []byte("a key from file"), cmd.GetRepositorySecret())
}

// ---------- Execute: location already prefixed with ptar: ----------

func TestCov2ExecutePtarSchemePrefix(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	src := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	tmpDir := t.TempDir()
	out := "ptar://" + filepath.Join(tmpDir, "scheme.ptar")

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{"-plaintext", "-o", out, filepath.Join(src, "subdir")}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// ---------- Execute: encrypted (default, no -plaintext) ----------

func TestCov2ExecuteEncrypted(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	src := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	tmpDir := t.TempDir()
	t.Setenv("PLAKAR_PASSPHRASE", "supersecretpass")

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-o", filepath.Join(tmpDir, "enc.ptar"), filepath.Join(src, "subdir"),
	}))
	require.False(t, cmd.NoEncryption)
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// ---------- Execute: @source resolved successfully in backup ----------

func TestCov2ExecuteAtSourceResolved(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	src := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	tmpDir := t.TempDir()

	ctx.Config = config.NewConfig()
	ctx.Config.Sources["mysrc"] = map[string]string{"location": "fs:" + filepath.Join(src, "subdir")}

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{"-plaintext", "-o", filepath.Join(tmpDir, "at.ptar")}))
	cmd.BackupTargets = listFlag{"@mysrc"}
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// ---------- Execute: overwrite cannot remove a directory ----------

func TestCov2ExecuteOverwriteRemoveFails(t *testing.T) {
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)
	src := ptesting.GenerateFiles(t, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	tmpDir := t.TempDir()
	// make the output path a non-empty directory so os.Remove fails
	out := filepath.Join(tmpDir, "isdir.ptar")
	require.NoError(t, os.MkdirAll(filepath.Join(out, "child"), 0755))

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, []string{"-plaintext", "-overwrite", "-o", out, filepath.Join(src, "subdir")}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "could not remove existing ptar archive")
}
