package pkg

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/location"
	ppkg "github.com/PlakarKorp/pkg"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Parse argument validation
// ---------------------------------------------------------------------------

func TestPkgParseNoAction(t *testing.T) {
	ctx := newCtx(t)
	// Pkg.Parse always reports "no action specified" for no args.
	err := (&Pkg{}).Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no action specified")

	// With a positional arg it reports "invalid argument".
	err = (&Pkg{}).Parse(ctx, []string{"bogus"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid argument")
}

func TestPkgAddParseVariants(t *testing.T) {
	ctx := newCtx(t)

	// -u (upgrade) with no positional args is allowed.
	cmd := &PkgAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"-u"}))
	require.True(t, cmd.upgrade)
	require.Empty(t, cmd.Args)

	// A relative name pointing to a real file gets absolutified.
	f := filepath.Join(ctx.CWD, "plugin.ptar")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0644))
	cmd = &PkgAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"plugin.ptar"}))
	require.Equal(t, []string{f}, cmd.Args)

	// A relative name that does not exist is kept as-is (treated as a recipe).
	cmd = &PkgAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"imap"}))
	require.Equal(t, []string{"imap"}, cmd.Args)

	// An absolute path that does not exist is an error.
	cmd = &PkgAdd{}
	missing := filepath.Join(ctx.CWD, "does-not-exist.ptar")
	err := cmd.Parse(ctx, []string{missing})
	require.Error(t, err)
	require.Contains(t, err.Error(), "file not found")
}

func TestPkgRmParseEmpty(t *testing.T) {
	ctx := newCtx(t)
	cmd := &PkgRm{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Empty(t, cmd.Args)
}

func TestPkgListParseTooMany(t *testing.T) {
	ctx := newCtx(t)
	err := (&PkgList{}).Parse(ctx, []string{"a", "b"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

func TestPkgBuildParseErrors(t *testing.T) {
	ctx := newCtx(t)

	// wrong number of args
	err := (&PkgBuild{}).Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong usage")

	// recipe file that does not exist (path contains separator -> file branch)
	bad := filepath.Join(ctx.CWD, "nope.yaml")
	err = (&PkgBuild{}).Parse(ctx, []string{bad})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse")

	// valid recipe but invalid plugin name
	recipe := filepath.Join(ctx.CWD, "recipe.yaml")
	require.NoError(t, os.WriteFile(recipe, []byte("name: \"bad name!\"\nversion: v1.0.0\nrepository: https://example.com/x\n"), 0644))
	err = (&PkgBuild{}).Parse(ctx, []string{recipe})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a valid plugin name")

	// valid name but invalid semver
	recipe2 := filepath.Join(ctx.CWD, "recipe2.yaml")
	require.NoError(t, os.WriteFile(recipe2, []byte("name: imap\nversion: not-semver\nrepository: https://example.com/x\n"), 0644))
	err = (&PkgBuild{}).Parse(ctx, []string{recipe2})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a valid version string")

	// fully valid recipe parses without error
	recipe3 := filepath.Join(ctx.CWD, "recipe3.yaml")
	require.NoError(t, os.WriteFile(recipe3, []byte("name: imap\nversion: v1.2.3\nrepository: https://example.com/x\n"), 0644))
	cmd := &PkgBuild{}
	require.NoError(t, cmd.Parse(ctx, []string{recipe3}))
	require.Equal(t, "imap", cmd.Recipe.Name)
	require.Equal(t, "v1.2.3", cmd.Recipe.Version)
}

func TestPkgCreateParseOk(t *testing.T) {
	ctx := newCtx(t)
	manifest := filepath.Join(ctx.CWD, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\nconnectors:\n  - type: importer\n    executable: myplugin\n"), 0644))

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"manifest.yaml", "v1.2.3"}))
	require.Equal(t, "myplugin", cmd.Manifest.Name)
	require.Equal(t, ctx.CWD, cmd.Base)
	require.Equal(t, manifest, cmd.ManifestPath)
	// Default output name is derived from the manifest name and version.
	require.True(t, strings.HasSuffix(cmd.Out, ".ptar"))
	require.Contains(t, filepath.Base(cmd.Out), "myplugin_v1.2.3_")

	// Explicit -out overrides the derived name.
	cmd = &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-out", "custom.ptar", "manifest.yaml", "v1.2.3"}))
	require.Equal(t, "custom.ptar", cmd.Out)
}

func TestPkgCreateParseBadManifestContent(t *testing.T) {
	ctx := newCtx(t)
	manifest := filepath.Join(ctx.CWD, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("\t this : is : not : valid : yaml\n"), 0644))
	err := (&PkgCreate{}).Parse(ctx, []string{"manifest.yaml", "v1.2.3"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse the manifest")
}

// ---------------------------------------------------------------------------
// getRecipe: file and http branches
// ---------------------------------------------------------------------------

func TestGetRecipeFile(t *testing.T) {
	ctx := newCtx(t)
	p := filepath.Join(ctx.CWD, "recipe.yaml")
	require.NoError(t, os.WriteFile(p, []byte("name: imap\nversion: v1.0.0\nrepository: https://example.com/r\n"), 0644))

	var r ppkg.Recipe
	require.NoError(t, getRecipe(ctx, p, &r))
	require.Equal(t, "imap", r.Name)
	require.Equal(t, "v1.0.0", r.Version)

	// non-existent file with a path separator -> open error
	var r2 ppkg.Recipe
	err := getRecipe(ctx, filepath.Join(ctx.CWD, "missing.yaml"), &r2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "couldn't open")
}

func TestGetRecipeHTTP(t *testing.T) {
	ctx := newCtx(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("name: s3\nversion: v2.0.0\nrepository: https://example.com/s3\n"))
	}))
	defer srv.Close()

	var r ppkg.Recipe
	require.NoError(t, getRecipe(ctx, srv.URL, &r))
	require.Equal(t, "s3", r.Name)
	require.Equal(t, "v2.0.0", r.Version)
}

func TestGetRecipeHTTPNotFound(t *testing.T) {
	ctx := newCtx(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var r ppkg.Recipe
	err := getRecipe(ctx, srv.URL, &r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "couldn't fetch recipe")
}

// ---------------------------------------------------------------------------
// absolutify
// ---------------------------------------------------------------------------

func TestAbsolutify(t *testing.T) {
	abs := "/already/abs/../abs/path"
	require.Equal(t, filepath.Clean(abs), absolutify("/base", abs))

	require.Equal(t, filepath.Join("/base", "rel", "file"), absolutify("/base", "rel/file"))
}

// ---------------------------------------------------------------------------
// pkgerImporter: metadata, Ping, Close, dofile, scan
// ---------------------------------------------------------------------------

func TestPkgerImporterMetadata(t *testing.T) {
	imp := &pkgerImporter{}
	require.Equal(t, "", imp.Origin())
	require.Equal(t, "pkger", imp.Type())
	require.Equal(t, "/", imp.Root())
	require.Equal(t, location.Flags(0), imp.Flags())
	require.NoError(t, imp.Ping(t.Context()))
	require.NoError(t, imp.Close(t.Context()))
}

func drain(ch <-chan *connectors.Record) []*connectors.Record {
	var recs []*connectors.Record
	for r := range ch {
		recs = append(recs, r)
	}
	return recs
}

func TestDofileExecutableOk(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "plugin")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755))

	imp := &pkgerImporter{cwd: dir}
	ch := make(chan *connectors.Record, 32)
	require.NoError(t, imp.dofile(exe, ch, itexe))
	close(ch)
	recs := drain(ch)

	var fileRec *connectors.Record
	for _, r := range recs {
		if r.Pathname == "/plugin" {
			fileRec = r
		}
		require.Nil(t, r.Err, "no record should carry an error")
	}
	require.NotNil(t, fileRec)
	require.NotNil(t, fileRec.Reader)
}

func TestDofileNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}
	dir := t.TempDir()
	plain := filepath.Join(dir, "data")
	require.NoError(t, os.WriteFile(plain, []byte("nope"), 0644))

	imp := &pkgerImporter{cwd: dir}
	ch := make(chan *connectors.Record, 32)
	err := imp.dofile(plain, ch, itexe)
	close(ch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Not executable")
}

func TestDofileJSONValidAndInvalid(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	require.NoError(t, os.WriteFile(good, []byte(`{"k":"v"}`), 0644))
	bad := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(bad, []byte(`{not json`), 0644))

	imp := &pkgerImporter{cwd: dir}

	ch := make(chan *connectors.Record, 32)
	require.NoError(t, imp.dofile(good, ch, itjson))
	close(ch)

	ch2 := make(chan *connectors.Record, 32)
	err := imp.dofile(bad, ch2, itjson)
	close(ch2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid json")
}

func TestDofileMissingFile(t *testing.T) {
	dir := t.TempDir()
	imp := &pkgerImporter{cwd: dir}
	ch := make(chan *connectors.Record, 32)
	err := imp.dofile(filepath.Join(dir, "ghost"), ch, itextra)
	close(ch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to open file")
}

func TestDofileNotBelowManifest(t *testing.T) {
	dir := t.TempDir()
	imp := &pkgerImporter{cwd: filepath.Join(dir, "sub")}
	require.NoError(t, os.MkdirAll(imp.cwd, 0755))
	outside := filepath.Join(dir, "outside.txt")
	require.NoError(t, os.WriteFile(outside, []byte("x"), 0644))

	ch := make(chan *connectors.Record, 32)
	err := imp.dofile(outside, ch, itextra)
	close(ch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not below the manifest")
}

func TestPkgerImporterScan(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\n"), 0644))
	exe := filepath.Join(dir, "myplugin")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755))
	validator := filepath.Join(dir, "validator.json")
	require.NoError(t, os.WriteFile(validator, []byte(`{"ok":true}`), 0644))
	extra := filepath.Join(dir, "extra.dat")
	require.NoError(t, os.WriteFile(extra, []byte("data"), 0644))

	m := &ppkg.Manifest{
		Name: "myplugin",
		Connectors: []ppkg.ManifestConnector{{
			Executable: "myplugin",
			Validator:  "validator.json",
			ExtraFiles: []string{"extra.dat"},
		}},
	}
	imp := &pkgerImporter{cwd: dir, manifest: m}

	ch := make(chan *connectors.Record, 128)
	require.NoError(t, imp.Import(context.Background(), ch, nil))
	recs := drain(ch)

	seen := map[string]bool{}
	for _, r := range recs {
		require.Nil(t, r.Err, "scan record %q carries error", r.Pathname)
		seen[r.Pathname] = true
	}
	require.True(t, seen["/manifest.yaml"])
	require.True(t, seen["/myplugin"])
	require.True(t, seen["/validator.json"])
	require.True(t, seen["/extra.dat"])
}

// ---------------------------------------------------------------------------
// Full PkgCreate.Parse + Execute: build a real .ptar
// ---------------------------------------------------------------------------

func TestPkgCreateExecuteBuildsPtar(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	work := t.TempDir()
	ctx.CWD = work

	manifest := filepath.Join(work, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\nconnectors:\n  - type: importer\n    executable: myplugin\n"), 0644))
	exe := filepath.Join(work, "myplugin")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\necho hi\n"), 0755))

	out := filepath.Join(work, "out.ptar")

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-out", out, "manifest.yaml", "v1.2.3"}))

	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	info, err := os.Stat(out)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
	require.Contains(t, bufOut.String(), "Plugin created successfully")
}
