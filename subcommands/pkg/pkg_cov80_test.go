package pkg

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// dofile itextra: a plain (non-exe, non-json) file is accepted as-is and a
// reader record is emitted. Exercises the itextra path (no type checks).
// ---------------------------------------------------------------------------

func TestDofileExtraPlainFile_cov80(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f := filepath.Join(dir, "data.bin")
	require.NoError(t, os.WriteFile(f, []byte("arbitrary-bytes"), 0644))

	imp := &pkgerImporter{cwd: dir}
	ch := make(chan *connectors.Record, 16)
	require.NoError(t, imp.dofile(f, ch, itextra))
	close(ch)

	var fileRec *connectors.Record
	for r := range ch {
		require.Nil(t, r.Err)
		if r.Pathname == "/data.bin" {
			fileRec = r
		}
	}
	require.NotNil(t, fileRec)
	require.NotNil(t, fileRec.Reader)
}

// ---------------------------------------------------------------------------
// dofile: stat failure path. Open a directory and request itexe — on most
// platforms open succeeds and stat reports a directory (not executable by the
// mode&0111 rule unless dir bits set). We instead drive the "Failed to open"
// branch with a path whose parent is a file (ENOTDIR), exercising the open
// error record while staying below cwd.
// ---------------------------------------------------------------------------

func TestDofileOpenErrorNotDir_cov80(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR semantics differ on windows")
	}
	dir := t.TempDir()
	// a regular file used as if it were a directory component
	notdir := filepath.Join(dir, "afile")
	require.NoError(t, os.WriteFile(notdir, []byte("x"), 0644))
	target := filepath.Join(notdir, "child") // afile/child -> ENOTDIR on open

	imp := &pkgerImporter{cwd: dir}
	ch := make(chan *connectors.Record, 16)
	err := imp.dofile(target, ch, itextra)
	close(ch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to open file")
}

// ---------------------------------------------------------------------------
// PkgCreate.Execute: storage.Create failure when the output path's directory
// does not exist. This drives the "failed to create the storage" error return
// without needing the network or a PkgManager.
// ---------------------------------------------------------------------------

func TestPkgCreateExecuteStorageCreateFails_cov80(t *testing.T) {
	t.Parallel()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	work := t.TempDir()
	ctx.CWD = work
	manifest := filepath.Join(work, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\nconnectors:\n  - type: importer\n    executable: myplugin\n"), 0644))

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"manifest.yaml", "v1.0.0"}))
	// Point output into a non-existent directory so ptar creation fails.
	cmd.Out = filepath.Join(work, "no", "such", "dir", "out.ptar")

	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// ---------------------------------------------------------------------------
// PkgCreate.Parse: GOARCH-only override (GOOS left to runtime) still affects
// the derived output name, exercising the goarchEnv branch independently.
// ---------------------------------------------------------------------------

func TestPkgCreateParseGOARCHOnly_cov80(t *testing.T) {
	ctx := newCtx(t)
	manifest := filepath.Join(ctx.CWD, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: arch\nconnectors:\n  - type: importer\n    executable: arch\n"), 0644))

	t.Setenv("GOOS", "")
	t.Setenv("GOARCH", "riscv64")

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"manifest.yaml", "v0.1.0"}))
	require.Contains(t, filepath.Base(cmd.Out), "_riscv64.ptar")
	require.Contains(t, filepath.Base(cmd.Out), runtime.GOOS)
}

// ---------------------------------------------------------------------------
// PkgAdd.Parse: multiple positional args where a mix of an existing regular
// file (absolutified) and a recipe name (kept) are handled in the loop.
// ---------------------------------------------------------------------------

func TestPkgAddParseMixedArgs_cov80(t *testing.T) {
	ctx := newCtx(t)
	f := filepath.Join(ctx.CWD, "local.ptar")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0644))

	cmd := &PkgAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"local.ptar", "remoterecipe"}))
	require.Equal(t, f, cmd.Args[0])
	require.Equal(t, "remoterecipe", cmd.Args[1])
}

// ---------------------------------------------------------------------------
// PkgBuild.Parse: a recipe with a valid name+version but supplied via the
// file branch where the recipe omits the repository still parses (repository
// is only used at build time). Covers the post-getRecipe validation success.
// ---------------------------------------------------------------------------

func TestPkgBuildParseValidNoRepo_cov80(t *testing.T) {
	ctx := newCtx(t)
	recipe := filepath.Join(ctx.CWD, "recipe.yaml")
	require.NoError(t, os.WriteFile(recipe, []byte("name: conn_1\nversion: v1.0.0\n"), 0644))
	cmd := &PkgBuild{}
	require.NoError(t, cmd.Parse(ctx, []string{recipe}))
	require.Equal(t, "conn_1", cmd.Recipe.Name)
}

// ---------------------------------------------------------------------------
// Pkg.Parse: invalid-argument branch (positional arg present).
// ---------------------------------------------------------------------------

func TestPkgParseInvalidArgument_cov80(t *testing.T) {
	ctx := newCtx(t)
	err := (&Pkg{}).Parse(ctx, []string{"frobnicate"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid argument: frobnicate")
}
