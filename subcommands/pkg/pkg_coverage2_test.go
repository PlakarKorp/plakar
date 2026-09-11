package pkg

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	ppkg "github.com/PlakarKorp/pkg"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Pkg.Execute / Pkg.Parse final branches
// ---------------------------------------------------------------------------

func TestPkgExecuteNoAction(t *testing.T) {
	ctx := newCtx(t)
	status, err := (&Pkg{}).Execute(ctx, nil)
	require.Equal(t, 1, status)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no action specified")
}

// ---------------------------------------------------------------------------
// PkgCreate.Parse: absolute manifest path + GOOS/GOARCH env derived name
// ---------------------------------------------------------------------------

func TestPkgCreateParseAbsoluteManifest(t *testing.T) {
	ctx := newCtx(t)
	manifest := filepath.Join(ctx.CWD, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\nconnectors:\n  - type: importer\n    executable: myplugin\n"), 0644))

	// Pass an absolute, uncleaned path to take the filepath.Clean branch.
	dirty := filepath.Join(ctx.CWD, ".", "manifest.yaml")
	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{dirty, "v1.0.0"}))
	require.Equal(t, manifest, cmd.ManifestPath)
	require.Equal(t, ctx.CWD, cmd.Base)
}

func TestPkgCreateParseGOOSGOARCHEnv(t *testing.T) {
	ctx := newCtx(t)
	manifest := filepath.Join(ctx.CWD, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\nconnectors:\n  - type: importer\n    executable: myplugin\n"), 0644))

	t.Setenv("GOOS", "plan9")
	t.Setenv("GOARCH", "mips")

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"manifest.yaml", "v3.4.5"}))
	// The derived output name must use the overridden GOOS/GOARCH.
	require.Contains(t, filepath.Base(cmd.Out), "myplugin_v3.4.5_plan9_mips.ptar")
}

// ---------------------------------------------------------------------------
// PkgCreate.Execute: failure when a connector file is not packageable.
// A connector pointing at a non-executable file makes the importer emit an
// error record, which makes Below.Errors != 0 and Execute return a failure.
// ---------------------------------------------------------------------------

func TestPkgCreateExecuteFailsOnBadConnector(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	work := t.TempDir()
	ctx.CWD = work

	// connector executable is NOT marked executable -> dofile emits an error
	// record during the scan.
	manifest := filepath.Join(work, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\nconnectors:\n  - type: importer\n    executable: myplugin\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(work, "myplugin"), []byte("not exec"), 0644))

	out := filepath.Join(work, "out.ptar")
	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-out", out, "manifest.yaml", "v1.0.0"}))

	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "failed to populate the snapshot: Not executable:")
}

// ---------------------------------------------------------------------------
// getRecipe: http transport error and malformed http body
// ---------------------------------------------------------------------------

func TestGetRecipeHTTPTransportError(t *testing.T) {
	ctx := newCtx(t)
	// A server that we immediately close yields a connection error from http.Get.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	var r ppkg.Recipe
	err := getRecipe(ctx, url, &r)
	require.Error(t, err)
}

func TestGetRecipeHTTPMalformedBody(t *testing.T) {
	ctx := newCtx(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("\t: this is : not : valid : yaml :\n"))
	}))
	defer srv.Close()

	var r ppkg.Recipe
	err := getRecipe(ctx, srv.URL, &r)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// PkgBuild.Parse: http recipe source feeding getRecipe's http branch
// ---------------------------------------------------------------------------

func TestPkgBuildParseFromHTTP(t *testing.T) {
	ctx := newCtx(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("name: imap\nversion: v2.3.4\nrepository: https://example.com/imap\n"))
	}))
	defer srv.Close()

	cmd := &PkgBuild{}
	require.NoError(t, cmd.Parse(ctx, []string{srv.URL}))
	require.Equal(t, "imap", cmd.Recipe.Name)
	require.Equal(t, "v2.3.4", cmd.Recipe.Version)
}

// ---------------------------------------------------------------------------
// PkgAdd.Parse: a directory (non-regular file) is kept as a recipe name, not
// absolutified, and does not error.
// ---------------------------------------------------------------------------

func TestPkgAddParseDirectoryKeptAsRecipe(t *testing.T) {
	ctx := newCtx(t)
	// A relative name that resolves to an existing *directory* is not a regular
	// file, so it is kept as-is (treated as a recipe name).
	require.NoError(t, os.Mkdir(filepath.Join(ctx.CWD, "adir"), 0755))
	cmd := &PkgAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"adir"}))
	require.Equal(t, []string{"adir"}, cmd.Args)
}

// ---------------------------------------------------------------------------
// pkgerImporter.dofile: windows .exe naming branch via GOOS env
// ---------------------------------------------------------------------------

func TestDofileWindowsExeBranch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GOOS", "windows")

	// A non-.exe file is "not executable" under the windows naming rule, even
	// though its mode bits are set.
	plain := filepath.Join(dir, "tool")
	require.NoError(t, os.WriteFile(plain, []byte("x"), 0755))
	imp := &pkgerImporter{cwd: dir}
	ch := make(chan *connectors.Record, 32)
	err := imp.dofile(plain, ch, itexe)
	close(ch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Not executable")

	// A .exe file is considered executable regardless of mode bits.
	exe := filepath.Join(dir, "tool.exe")
	require.NoError(t, os.WriteFile(exe, []byte("x"), 0644))
	ch2 := make(chan *connectors.Record, 32)
	require.NoError(t, imp.dofile(exe, ch2, itexe))
	close(ch2)
	for _, r := range drainRecords(ch2) {
		require.Nil(t, r.Err)
	}
}

func drainRecords(ch <-chan *connectors.Record) []*connectors.Record {
	var recs []*connectors.Record
	for r := range ch {
		recs = append(recs, r)
	}
	return recs
}

// ---------------------------------------------------------------------------
// pkgerImporter.scan: error propagation when a connector executable is missing.
// Import returns an error for the missing file; when the manifest path itself
// is below cwd this still walks the connector loop.
// ---------------------------------------------------------------------------

func TestScanEmitsErrorForMissingConnectorFiles(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\n"), 0644))

	m := &ppkg.Manifest{
		Name: "myplugin",
		Connectors: []ppkg.ManifestConnector{{
			Executable: "ghost-exe",                 // missing
			Validator:  "ghost-validator.json",      // missing
			ExtraFiles: []string{"ghost-extra.dat"}, // missing
		}},
	}
	imp := &pkgerImporter{cwd: dir, manifest: m}

	ch := make(chan *connectors.Record, 128)
	err := imp.Import(context.Background(), ch, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
}

// TestScanPropagatesNotBelowManifestError checks scan returns the hard error
// (not just an error record) when a connector executable escapes the manifest
// directory.
func TestScanPropagatesNotBelowManifestError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("path-escape semantics differ on windows")
	}
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	require.NoError(t, os.Mkdir(sub, 0755))
	manifest := filepath.Join(sub, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifest, []byte("name: myplugin\n"), 0644))

	// Absolute executable path outside cwd triggers the "not below the manifest"
	// hard error from dofile, which scan returns directly.
	outside := filepath.Join(root, "outside")
	require.NoError(t, os.WriteFile(outside, []byte("#!/bin/sh\n"), 0755))

	m := &ppkg.Manifest{
		Name: "myplugin",
		Connectors: []ppkg.ManifestConnector{{
			Executable: outside,
		}},
	}
	imp := &pkgerImporter{cwd: sub, manifest: m}

	ch := make(chan *connectors.Record, 64)
	err := imp.Import(context.Background(), ch, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not below the manifest")
}
