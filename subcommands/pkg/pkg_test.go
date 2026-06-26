package pkg

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

// newCtx returns a minimal AppContext suitable for Parse() tests that don't
// touch the package manager or storage. CWD is set to a temp dir.
func newCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.CWD = t.TempDir()
	return ctx
}

// ---------------------------------------------------------------------------
// Pkg (dispatcher) Parse / Execute
// ---------------------------------------------------------------------------

func TestPkgParse(t *testing.T) {
	ctx := newCtx(t)

	// no action specified
	cmd := &Pkg{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no action specified")

	// invalid argument
	cmd = &Pkg{}
	err = cmd.Parse(ctx, []string{"bogus"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid argument")
}

func TestPkgExecute(t *testing.T) {
	ctx := newCtx(t)
	cmd := &Pkg{}
	status, err := cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// ---------------------------------------------------------------------------
// PkgAdd.Parse
// ---------------------------------------------------------------------------

func TestPkgAddParse(t *testing.T) {
	ctx := newCtx(t)

	// not enough arguments without -u
	cmd := &PkgAdd{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not enough arguments")

	// -u with no args is allowed
	cmd = &PkgAdd{}
	err = cmd.Parse(ctx, []string{"-u"})
	require.NoError(t, err)
	require.True(t, cmd.upgrade)

	// a recipe name (not a file) passes through verbatim
	cmd = &PkgAdd{}
	err = cmd.Parse(ctx, []string{"imap"})
	require.NoError(t, err)
	require.Equal(t, []string{"imap"}, cmd.Args)

	// an existing relative file is absolutified
	f := filepath.Join(ctx.CWD, "plugin.ptar")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0644))
	cmd = &PkgAdd{}
	err = cmd.Parse(ctx, []string{"plugin.ptar"})
	require.NoError(t, err)
	require.Equal(t, []string{f}, cmd.Args)

	// an absolute path that does not exist is an error
	missing := filepath.Join(ctx.CWD, "does-not-exist.ptar")
	cmd = &PkgAdd{}
	err = cmd.Parse(ctx, []string{missing})
	require.Error(t, err)
	require.Contains(t, err.Error(), "file not found")
}

// ---------------------------------------------------------------------------
// PkgRm.Parse
// ---------------------------------------------------------------------------

func TestPkgRmParse(t *testing.T) {
	ctx := newCtx(t)

	cmd := &PkgRm{}
	err := cmd.Parse(ctx, []string{})
	require.NoError(t, err)
	require.Empty(t, cmd.Args)

	cmd = &PkgRm{}
	err = cmd.Parse(ctx, []string{"foo", "bar"})
	require.NoError(t, err)
	require.Equal(t, []string{"foo", "bar"}, cmd.Args)
}

// ---------------------------------------------------------------------------
// PkgList.Parse
// ---------------------------------------------------------------------------

func TestPkgListParse(t *testing.T) {
	ctx := newCtx(t)

	cmd := &PkgList{}
	err := cmd.Parse(ctx, []string{})
	require.NoError(t, err)
	require.False(t, cmd.LongName)
	require.False(t, cmd.ListAll)

	cmd = &PkgList{}
	err = cmd.Parse(ctx, []string{"-long", "-available"})
	require.NoError(t, err)
	require.True(t, cmd.LongName)
	require.True(t, cmd.ListAll)

	cmd = &PkgList{}
	err = cmd.Parse(ctx, []string{"extra"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

// ---------------------------------------------------------------------------
// PkgBuild.Parse (recipe validation)
// ---------------------------------------------------------------------------

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0644))
	return p
}

func TestPkgBuildParseWrongUsage(t *testing.T) {
	ctx := newCtx(t)
	cmd := &PkgBuild{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong usage")

	cmd = &PkgBuild{}
	err = cmd.Parse(ctx, []string{"a.yaml", "b.yaml"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong usage")
}

func TestPkgBuildParseRecipeFromFile(t *testing.T) {
	ctx := newCtx(t)

	// valid recipe (path contains a separator -> file branch of getRecipe)
	recipe := writeFile(t, ctx.CWD, "recipe.yaml",
		"name: myplugin\nversion: v1.0.0\nrepository: https://example.com/x.git\n")
	cmd := &PkgBuild{}
	err := cmd.Parse(ctx, []string{recipe})
	require.NoError(t, err)
	require.Equal(t, "myplugin", cmd.Recipe.Name)
	require.Equal(t, "v1.0.0", cmd.Recipe.Version)

	// invalid plugin name
	badname := writeFile(t, ctx.CWD, "badname.yaml",
		"name: bad name!\nversion: v1.0.0\nrepository: https://example.com/x.git\n")
	cmd = &PkgBuild{}
	err = cmd.Parse(ctx, []string{badname})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a valid plugin name")

	// invalid version string
	badver := writeFile(t, ctx.CWD, "badver.yaml",
		"name: myplugin\nversion: notsemver\nrepository: https://example.com/x.git\n")
	cmd = &PkgBuild{}
	err = cmd.Parse(ctx, []string{badver})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a valid version string")

	// nonexistent recipe path (contains a separator -> file open fails)
	cmd = &PkgBuild{}
	err = cmd.Parse(ctx, []string{filepath.Join(ctx.CWD, "nope.yaml")})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse")
}

// ---------------------------------------------------------------------------
// getRecipe path branches
// ---------------------------------------------------------------------------

func TestGetRecipeFileBranch(t *testing.T) {
	ctx := newCtx(t)

	// absolute path
	p := writeFile(t, ctx.CWD, "r.yaml",
		"name: x\nversion: v1.0.0\nrepository: https://example.com/x.git\n")
	var r pkg.Recipe
	err := getRecipe(ctx, p, &r)
	require.NoError(t, err)
	require.Equal(t, "x", r.Name)

	// path with a separator that does not exist
	var r2 pkg.Recipe
	err = getRecipe(ctx, filepath.Join(ctx.CWD, "missing.yaml"), &r2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "couldn't open")
}

func TestGetRecipeHTTPBranch(t *testing.T) {
	ctx := newCtx(t)

	// successful fetch over http
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "name: web\nversion: v2.0.0\nrepository: https://example.com/x.git\n")
	}))
	defer ok.Close()

	var r pkg.Recipe
	err := getRecipe(ctx, ok.URL, &r)
	require.NoError(t, err)
	require.Equal(t, "web", r.Name)
	require.Equal(t, "v2.0.0", r.Version)

	// non-200 response
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer bad.Close()

	var r2 pkg.Recipe
	err = getRecipe(ctx, bad.URL, &r2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "couldn't fetch recipe")
}

func TestPkgBuildParseRecipeFromHTTP(t *testing.T) {
	ctx := newCtx(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "name: web\nversion: v2.0.0\nrepository: https://example.com/x.git\n")
	}))
	defer srv.Close()

	cmd := &PkgBuild{}
	err := cmd.Parse(ctx, []string{srv.URL})
	require.NoError(t, err)
	require.Equal(t, "web", cmd.Recipe.Name)
}

// ---------------------------------------------------------------------------
// PkgCreate.Parse argument validation
// ---------------------------------------------------------------------------

func TestPkgCreateParseValidation(t *testing.T) {
	ctx := newCtx(t)

	// wrong usage (need exactly 2 args)
	cmd := &PkgCreate{}
	err := cmd.Parse(ctx, []string{"manifest.yaml"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "wrong usage")

	// bad version string
	cmd = &PkgCreate{}
	err = cmd.Parse(ctx, []string{"manifest.yaml", "notsemver"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad version string")

	// file name must be manifest.yaml
	other := writeFile(t, ctx.CWD, "other.yaml", "name: x\n")
	cmd = &PkgCreate{}
	err = cmd.Parse(ctx, []string{other, "v1.0.0"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "manifest.yaml")

	// manifest does not exist
	cmd = &PkgCreate{}
	err = cmd.Parse(ctx, []string{filepath.Join(ctx.CWD, "manifest.yaml"), "v1.0.0"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't open")
}

func TestPkgCreateParseDefaultOut(t *testing.T) {
	ctx := newCtx(t)
	dir := t.TempDir()
	manifest := writeFile(t, dir, "manifest.yaml",
		"name: myplugin\nversion: v1.0.0\nconnectors: []\n")

	cmd := &PkgCreate{}
	err := cmd.Parse(ctx, []string{manifest, "v1.0.0"})
	require.NoError(t, err)
	require.Equal(t, dir, cmd.Base)
	require.Equal(t, manifest, cmd.ManifestPath)
	// default out name uses the manifest name + version + GOOS/GOARCH
	expected := filepath.Join(ctx.CWD,
		"myplugin_v1.0.0_"+runtime.GOOS+"_"+runtime.GOARCH+".ptar")
	require.Equal(t, expected, cmd.Out)
}

func TestPkgCreateParseExplicitOut(t *testing.T) {
	ctx := newCtx(t)
	dir := t.TempDir()
	manifest := writeFile(t, dir, "manifest.yaml",
		"name: myplugin\nversion: v1.0.0\nconnectors: []\n")

	out := filepath.Join(t.TempDir(), "explicit.ptar")
	cmd := &PkgCreate{}
	err := cmd.Parse(ctx, []string{"-out", out, manifest, "v1.0.0"})
	require.NoError(t, err)
	require.Equal(t, out, cmd.Out)
}

// ---------------------------------------------------------------------------
// PkgCreate full Parse + Execute building a real .ptar
// ---------------------------------------------------------------------------

func TestPkgCreateExecuteBuildsPtar(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	// Build a plugin source tree: manifest + a fake executable connector.
	srcdir := t.TempDir()
	exe := filepath.Join(srcdir, "myconnector")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\necho hi\n"), 0755))
	manifest := writeFile(t, srcdir, "manifest.yaml",
		"name: myplugin\n"+
			"version: v1.0.0\n"+
			"connectors:\n"+
			"  - type: importer\n"+
			"    executable: myconnector\n")

	out := filepath.Join(t.TempDir(), "myplugin.ptar")

	cmd := &PkgCreate{}
	err := cmd.Parse(ctx, []string{"-out", out, manifest, "v1.0.0"})
	require.NoError(t, err)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// the .ptar was created and is non-empty
	info, err := os.Stat(out)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))

	require.Contains(t, bufOut.String(), "Plugin created successfully")
}

func TestPkgCreateExecuteNonExecutableFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	srcdir := t.TempDir()
	// connector file is NOT executable -> Backup should record an error
	notexe := filepath.Join(srcdir, "myconnector")
	require.NoError(t, os.WriteFile(notexe, []byte("nope"), 0644))
	manifest := writeFile(t, srcdir, "manifest.yaml",
		"name: myplugin\n"+
			"version: v1.0.0\n"+
			"connectors:\n"+
			"  - type: importer\n"+
			"    executable: myconnector\n")

	out := filepath.Join(t.TempDir(), "myplugin.ptar")
	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-out", out, manifest, "v1.0.0"}))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "failed to package all the files")
}

// ---------------------------------------------------------------------------
// Lookup wiring for the subcommands (registration coverage)
// ---------------------------------------------------------------------------

func TestPkgLookup(t *testing.T) {
	for _, tc := range []struct {
		args []string
	}{
		{[]string{"pkg"}},
		{[]string{"pkg", "add"}},
		{[]string{"pkg", "rm"}},
		{[]string{"pkg", "list"}},
		{[]string{"pkg", "show"}},
		{[]string{"pkg", "create"}},
		{[]string{"pkg", "build"}},
	} {
		sub, _, _ := subcommands.Lookup(tc.args)
		require.NotNil(t, sub, "lookup %v", tc.args)
	}
}

// ---------------------------------------------------------------------------
// pkgerImporter pure helpers
// ---------------------------------------------------------------------------

func TestAbsolutify(t *testing.T) {
	require.Equal(t, "/a/b", absolutify("/cwd", "/a/b/"))
	require.Equal(t, filepath.Join("/cwd", "rel"), absolutify("/cwd", "rel"))
}

func TestPkgerImporterMeta(t *testing.T) {
	imp := &pkgerImporter{}
	require.Equal(t, "", imp.Origin())
	require.Equal(t, "pkger", imp.Type())
	require.Equal(t, "/", imp.Root())
	require.Equal(t, 0, int(imp.Flags()))
	require.NoError(t, imp.Ping(nil))
	require.NoError(t, imp.Close(nil))
}

func drain(ch <-chan *connectors.Record) []*connectors.Record {
	var recs []*connectors.Record
	for r := range ch {
		recs = append(recs, r)
	}
	return recs
}

func TestPkgerImporterScan(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}
	srcdir := t.TempDir()
	exe := filepath.Join(srcdir, "myconnector")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755))
	validator := filepath.Join(srcdir, "validator.json")
	require.NoError(t, os.WriteFile(validator, []byte(`{"a":1}`), 0644))
	extra := filepath.Join(srcdir, "extra.txt")
	require.NoError(t, os.WriteFile(extra, []byte("data"), 0644))
	manifest := writeFile(t, srcdir, "manifest.yaml", "name: x\nversion: v1.0.0\n")

	imp := &pkgerImporter{
		cwd:          srcdir,
		manifestPath: manifest,
		manifest: &pkg.Manifest{
			Name: "x",
			Connectors: []pkg.ManifestConnector{
				{
					Executable: "myconnector",
					Validator:  "validator.json",
					ExtraFiles: []string{"extra.txt"},
				},
			},
		},
	}

	ch := make(chan *connectors.Record, 64)
	err := imp.scan(ch)
	close(ch)
	require.NoError(t, err)

	recs := drain(ch)
	names := map[string]bool{}
	for _, r := range recs {
		names[r.Pathname] = true
	}
	require.True(t, names["/"])
	require.True(t, names["/manifest.yaml"])
	require.True(t, names["/myconnector"])
	require.True(t, names["/validator.json"])
	require.True(t, names["/extra.txt"])
}

func TestPkgerImporterScanNonExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}
	srcdir := t.TempDir()
	notexe := filepath.Join(srcdir, "myconnector")
	require.NoError(t, os.WriteFile(notexe, []byte("nope"), 0644))
	manifest := writeFile(t, srcdir, "manifest.yaml", "name: x\nversion: v1.0.0\n")

	imp := &pkgerImporter{
		cwd:          srcdir,
		manifestPath: manifest,
		manifest: &pkg.Manifest{
			Connectors: []pkg.ManifestConnector{{Executable: "myconnector"}},
		},
	}

	ch := make(chan *connectors.Record, 64)
	err := imp.scan(ch)
	close(ch)
	require.NoError(t, err)

	recs := drain(ch)
	var sawErr bool
	for _, r := range recs {
		if r.Err != nil {
			sawErr = true
		}
	}
	require.True(t, sawErr, "expected an error record for the non-executable")
}

func TestPkgerImporterScanInvalidJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable bit semantics differ on windows")
	}
	srcdir := t.TempDir()
	exe := filepath.Join(srcdir, "myconnector")
	require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0755))
	validator := filepath.Join(srcdir, "validator.json")
	require.NoError(t, os.WriteFile(validator, []byte("not json"), 0644))
	manifest := writeFile(t, srcdir, "manifest.yaml", "name: x\nversion: v1.0.0\n")

	imp := &pkgerImporter{
		cwd:          srcdir,
		manifestPath: manifest,
		manifest: &pkg.Manifest{
			Connectors: []pkg.ManifestConnector{
				{Executable: "myconnector", Validator: "validator.json"},
			},
		},
	}

	ch := make(chan *connectors.Record, 64)
	err := imp.scan(ch)
	close(ch)
	require.NoError(t, err)

	recs := drain(ch)
	var sawErr bool
	for _, r := range recs {
		if r.Err != nil {
			sawErr = true
		}
	}
	require.True(t, sawErr, "expected an error record for invalid json")
}

func TestPkgerImporterDofileMissing(t *testing.T) {
	srcdir := t.TempDir()
	imp := &pkgerImporter{cwd: srcdir, manifest: &pkg.Manifest{}}

	ch := make(chan *connectors.Record, 8)
	err := imp.dofile(filepath.Join(srcdir, "nope"), ch, itextra)
	close(ch)
	require.NoError(t, err)

	recs := drain(ch)
	require.Len(t, recs, 1)
	require.NotNil(t, recs[0].Err)
}

func TestPkgerImporterDofileEscape(t *testing.T) {
	srcdir := t.TempDir()
	imp := &pkgerImporter{cwd: srcdir, manifest: &pkg.Manifest{}}

	// a file that resolves outside the manifest dir must be rejected
	outside := filepath.Join(filepath.Dir(srcdir), "outside.txt")
	require.NoError(t, os.WriteFile(outside, []byte("x"), 0644))

	ch := make(chan *connectors.Record, 8)
	err := imp.dofile(outside, ch, itextra)
	close(ch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not below the manifest")
}

func TestMkstruct(t *testing.T) {
	ch := make(chan *connectors.Record, 16)
	mkstruct("/a/b/c/file.txt", ch)
	close(ch)
	recs := drain(ch)
	got := map[string]bool{}
	for _, r := range recs {
		got[r.Pathname] = true
	}
	require.True(t, got["/a/b/c"])
	require.True(t, got["/a/b"])
	require.True(t, got["/a"])
	require.False(t, got["/"])
}
