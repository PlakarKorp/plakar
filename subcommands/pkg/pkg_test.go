package pkg

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	ppkg "github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func newCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bytes.NewBuffer(nil)
	ctx.Stderr = bytes.NewBuffer(nil)
	ctx.CWD = t.TempDir()
	return ctx
}

func TestPkgRegisteredFactories(t *testing.T) {
	cases := []struct {
		args []string
		typ  interface{}
	}{
		{[]string{"pkg", "add"}, &PkgAdd{}},
		{[]string{"pkg", "rm"}, &PkgRm{}},
		{[]string{"pkg", "create"}, &PkgCreate{}},
		{[]string{"pkg", "build"}, &PkgBuild{}},
		{[]string{"pkg", "list"}, &PkgList{}},
		{[]string{"pkg", "show"}, &PkgList{}},
	}
	for _, c := range cases {
		cmd, _, _ := subcommands.Lookup(c.args)
		require.NotNil(t, cmd, "args=%v", c.args)
		require.IsType(t, c.typ, cmd)
	}
}

func TestPkgListParse(t *testing.T) {
	ctx := newCtx(t)
	cmd := &PkgList{}
	require.NoError(t, cmd.Parse(ctx, []string{}))

	cmd = &PkgList{}
	require.NoError(t, cmd.Parse(ctx, []string{"-long", "-available"}))
	require.True(t, cmd.LongName)
	require.True(t, cmd.ListAll)

	// extra positional argument is rejected
	require.Error(t, (&PkgList{}).Parse(ctx, []string{"extra"}))
}

func TestPkgCreateParseErrors(t *testing.T) {
	ctx := newCtx(t)

	// wrong arg count
	require.Error(t, (&PkgCreate{}).Parse(ctx, []string{"manifest.yaml"}))

	// bad version string
	require.Error(t, (&PkgCreate{}).Parse(ctx, []string{"manifest.yaml", "not-a-semver"}))

	// manifest filename must be exactly "manifest.yaml"
	err := (&PkgCreate{}).Parse(ctx, []string{"wrong-name.yaml", "v1.2.3"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "manifest.yaml")

	// correctly-named but missing manifest file: open fails
	err = (&PkgCreate{}).Parse(ctx, []string{"manifest.yaml", "v1.2.3"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't open")
}

func TestPkgRmParse(t *testing.T) {
	ctx := newCtx(t)
	// rm accepts a list of plugin names (empty is allowed at parse time).
	cmd := &PkgRm{}
	require.NoError(t, cmd.Parse(ctx, []string{"plugin-a", "plugin-b"}))
	require.Equal(t, []string{"plugin-a", "plugin-b"}, cmd.Args)
}

func TestPkgAddParse(t *testing.T) {
	ctx := newCtx(t)
	// add with no package name should error.
	require.Error(t, (&PkgAdd{}).Parse(ctx, []string{}))
}

// TestPkgAddLatestResolvesThroughRecipe checks that the "@latest" suffix is the
// explicit spelling of the default version: it must let the package manager
// resolve the version from the recipe, not be forwarded as a literal version.
func TestPkgAddLatestResolvesThroughRecipe(t *testing.T) {
	var mu sync.Mutex
	var requested []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requested = append(requested, r.URL.Path)
		mu.Unlock()
		if r.URL.Path == path.Join("/community/", ppkg.PLUGIN_API_VERSION, "s3", "recipe.yaml") {
			w.Write([]byte("name: s3\nversion: v2.0.0\nrepository: https://example.com/s3\n"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ctx := newCtx(t)
	backend, err := ppkg.NewFlatBackend(ctx.GetInner(),
		filepath.Join(ctx.CWD, "plugins"), filepath.Join(ctx.CWD, "cache"),
		&ppkg.FlatBackendOptions{})
	require.NoError(t, err)

	manager, err := ppkg.New(backend, &ppkg.Options{DistURL: srv.URL})
	require.NoError(t, err)
	ctx.SetPkgManager(manager)

	cmd := &PkgAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"s3@latest"}))
	// The download itself cannot succeed against this stub server; what matters
	// is which artifact the manager was asked for.
	_, _ = cmd.Execute(ctx, nil)

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, requested)
	require.Equal(t, path.Join("/community/", ppkg.PLUGIN_API_VERSION, "s3", "recipe.yaml"), requested[0])
	for _, p := range requested {
		require.NotContains(t, p, "latest",
			"@latest must not be forwarded as a literal version")
	}
}
