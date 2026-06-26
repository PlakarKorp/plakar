package pkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/pkg"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// getRecipe: http transport error and malformed-body parse error
// ---------------------------------------------------------------------------

func TestGetRecipeHTTPTransportError(t *testing.T) {
	ctx := newCtx(t)
	// nothing listens on this port -> http.Get returns a transport error
	var r pkg.Recipe
	err := getRecipe(ctx, "http://127.0.0.1:1/recipe.yaml", &r)
	require.Error(t, err)
}

func TestGetRecipeHTTPMalformedBody(t *testing.T) {
	ctx := newCtx(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// invalid yaml content so recipe.Parse fails
		w.Write([]byte("\t: : not valid yaml : :\n  - broken"))
	}))
	defer srv.Close()

	var r pkg.Recipe
	err := getRecipe(ctx, srv.URL, &r)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// PkgBuild.Parse: recipe with an empty name fails the namere check
// ---------------------------------------------------------------------------

func TestPkgBuildParseEmptyName(t *testing.T) {
	ctx := newCtx(t)
	recipe := writeFile(t, ctx.CWD, "empty.yaml",
		"version: v1.0.0\nrepository: https://example.com/x.git\n")
	cmd := &PkgBuild{}
	err := cmd.Parse(ctx, []string{recipe})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a valid plugin name")
}

func TestPkgBuildParseHTTPMalformed(t *testing.T) {
	ctx := newCtx(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("\t: : broken"))
	}))
	defer srv.Close()

	cmd := &PkgBuild{}
	err := cmd.Parse(ctx, []string{srv.URL})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse")
}

// ---------------------------------------------------------------------------
// PkgCreate.Parse: GOOS/GOARCH env overrides shape the default output name
// ---------------------------------------------------------------------------

func TestPkgCreateParseGOOSGOARCHOverride(t *testing.T) {
	ctx := newCtx(t)
	dir := t.TempDir()
	manifest := writeFile(t, dir, "manifest.yaml",
		"name: myplugin\nversion: v1.0.0\nconnectors: []\n")

	t.Setenv("GOOS", "plan9")
	t.Setenv("GOARCH", "mips")

	cmd := &PkgCreate{}
	err := cmd.Parse(ctx, []string{manifest, "v1.0.0"})
	require.NoError(t, err)

	expected := filepath.Join(ctx.CWD, "myplugin_v1.0.0_plan9_mips.ptar")
	require.Equal(t, expected, cmd.Out)
}

// PkgCreate.Parse with an absolute manifest path goes through the
// filepath.Clean branch rather than Join(CWD, ...).
func TestPkgCreateParseAbsoluteManifest(t *testing.T) {
	ctx := newCtx(t)
	dir := t.TempDir()
	manifest := writeFile(t, dir, "manifest.yaml",
		"name: myplugin\nversion: v1.0.0\nconnectors: []\n")

	// pass a non-clean absolute path to exercise filepath.Clean
	messy := filepath.Join(dir, ".", "manifest.yaml")
	cmd := &PkgCreate{}
	err := cmd.Parse(ctx, []string{messy, "v1.0.0"})
	require.NoError(t, err)
	require.Equal(t, manifest, cmd.ManifestPath)
	require.Equal(t, dir, cmd.Base)
}

// PkgCreate.Parse: a manifest.yaml whose contents are invalid fails at parse.
func TestPkgCreateParseBadManifest(t *testing.T) {
	ctx := newCtx(t)
	dir := t.TempDir()
	manifest := writeFile(t, dir, "manifest.yaml", "\t: : not valid : :")
	cmd := &PkgCreate{}
	err := cmd.Parse(ctx, []string{manifest, "v1.0.0"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse the manifest")
}

// ---------------------------------------------------------------------------
// pkgerImporter.Import drives scan and closes the channel
// ---------------------------------------------------------------------------

func TestPkgerImporterImport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}
	srcdir := t.TempDir()
	exe := filepath.Join(srcdir, "myconnector")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755))
	manifest := writeFile(t, srcdir, "manifest.yaml", "name: x\nversion: v1.0.0\n")

	imp := &pkgerImporter{
		cwd:          srcdir,
		manifestPath: manifest,
		manifest: &pkg.Manifest{
			Connectors: []pkg.ManifestConnector{{Executable: "myconnector"}},
		},
	}

	records := make(chan *connectors.Record, 64)
	// Import closes the records channel itself.
	err := imp.Import(context.Background(), records, nil)
	require.NoError(t, err)

	recs := drain(records)
	names := map[string]bool{}
	for _, r := range recs {
		names[r.Pathname] = true
	}
	require.True(t, names["/"])
	require.True(t, names["/manifest.yaml"])
	require.True(t, names["/myconnector"])
}

// Import surfaces the error when a connector executable is missing entirely:
// dofile records an error record but returns nil, so Import returns nil and
// the error is carried in-band on the record.
func TestPkgerImporterImportMissingExe(t *testing.T) {
	srcdir := t.TempDir()
	manifest := writeFile(t, srcdir, "manifest.yaml", "name: x\nversion: v1.0.0\n")

	imp := &pkgerImporter{
		cwd:          srcdir,
		manifestPath: manifest,
		manifest: &pkg.Manifest{
			Connectors: []pkg.ManifestConnector{{Executable: "nope"}},
		},
	}

	records := make(chan *connectors.Record, 64)
	err := imp.Import(context.Background(), records, nil)
	require.NoError(t, err)

	var sawErr bool
	for _, r := range drain(records) {
		if r.Err != nil {
			sawErr = true
		}
	}
	require.True(t, sawErr)
}

// ---------------------------------------------------------------------------
// dofile with the extra-files / validator paths through a connector
// ---------------------------------------------------------------------------

func TestPkgerImporterScanExtraMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}
	srcdir := t.TempDir()
	exe := filepath.Join(srcdir, "myconnector")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755))
	manifest := writeFile(t, srcdir, "manifest.yaml", "name: x\nversion: v1.0.0\n")

	imp := &pkgerImporter{
		cwd:          srcdir,
		manifestPath: manifest,
		manifest: &pkg.Manifest{
			Connectors: []pkg.ManifestConnector{
				{Executable: "myconnector", ExtraFiles: []string{"missing.txt"}},
			},
		},
	}

	ch := make(chan *connectors.Record, 64)
	err := imp.scan(ch)
	close(ch)
	require.NoError(t, err)

	var sawErr bool
	for _, r := range drain(ch) {
		if r.Err != nil {
			sawErr = true
		}
	}
	require.True(t, sawErr, "expected error record for the missing extra file")
}

// dofile rejects an extra file that escapes the manifest dir, and scan
// propagates that error.
func TestPkgerImporterScanExtraEscapes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}
	srcdir := t.TempDir()
	exe := filepath.Join(srcdir, "myconnector")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755))
	manifest := writeFile(t, srcdir, "manifest.yaml", "name: x\nversion: v1.0.0\n")

	outside := filepath.Join(filepath.Dir(srcdir), "escape.txt")
	require.NoError(t, os.WriteFile(outside, []byte("x"), 0644))

	imp := &pkgerImporter{
		cwd:          srcdir,
		manifestPath: manifest,
		manifest: &pkg.Manifest{
			Connectors: []pkg.ManifestConnector{
				{Executable: "myconnector", ExtraFiles: []string{outside}},
			},
		},
	}

	ch := make(chan *connectors.Record, 64)
	err := imp.scan(ch)
	close(ch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not below the manifest")
}

// ---------------------------------------------------------------------------
// clone: with PLAKAR_CLONE_TOKEN set, a repository URL that cannot be parsed
// fails before git is ever invoked.
// ---------------------------------------------------------------------------

func TestCloneTokenURLParseError(t *testing.T) {
	t.Setenv("PLAKAR_CLONE_TOKEN", "secrettoken")

	// a control byte makes url.Parse fail
	recipe := &pkg.Recipe{
		Name:       "x",
		Version:    "v1.0.0",
		Repository: "http://example.com/\x7frepo.git",
	}
	err := clone(t.TempDir(), recipe)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse repository URL")
}

// dofile with an absolute in-tree path exercises the absolutify abs branch.
func TestPkgerImporterDofileAbsolute(t *testing.T) {
	srcdir := t.TempDir()
	f := filepath.Join(srcdir, "data.txt")
	require.NoError(t, os.WriteFile(f, []byte("hello"), 0644))

	imp := &pkgerImporter{cwd: srcdir, manifest: &pkg.Manifest{}}
	ch := make(chan *connectors.Record, 8)
	err := imp.dofile(f, ch, itextra)
	close(ch)
	require.NoError(t, err)

	recs := drain(ch)
	var found bool
	for _, r := range recs {
		if r.Pathname == "/data.txt" && r.Err == nil {
			found = true
		}
	}
	require.True(t, found)
}
