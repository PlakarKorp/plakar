package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	ppkg "github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/stretchr/testify/require"
)

func newRegistryCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := newCtx(t)
	ctx.ConfigDir = t.TempDir()
	ctx.Config = config.NewConfig()
	return ctx
}

func runRegistry(t *testing.T, ctx *appcontext.AppContext, args ...string) error {
	t.Helper()
	cmd, _, rest := subcommands.Lookup(append([]string{"pkg", "registry"}, args...))
	require.NotNil(t, cmd)
	if err := cmd.Parse(ctx, rest); err != nil {
		return err
	}
	_, err := cmd.Execute(ctx, nil)
	return err
}

func stdout(ctx *appcontext.AppContext) string {
	out := ctx.Stdout.(*bytes.Buffer).String()
	ctx.Stdout.(*bytes.Buffer).Reset()
	return out
}

func stderr(ctx *appcontext.AppContext) string {
	out := ctx.Stderr.(*bytes.Buffer).String()
	ctx.Stderr.(*bytes.Buffer).Reset()
	return out
}

func TestPkgRegistryRegisteredFactories(t *testing.T) {
	cases := []struct {
		args []string
		typ  any
	}{
		{[]string{"pkg", "registry"}, &PkgRegistry{}},
		{[]string{"pkg", "registry", "add"}, &PkgRegistryAdd{}},
		{[]string{"pkg", "registry", "rm"}, &PkgRegistryRm{}},
		{[]string{"pkg", "registry", "list"}, &PkgRegistryList{}},
	}
	for _, c := range cases {
		cmd, _, _ := subcommands.Lookup(c.args)
		require.NotNil(t, cmd, "args=%v", c.args)
		require.IsType(t, c.typ, cmd)
	}
}

func TestPkgRegistryNoAction(t *testing.T) {
	err := runRegistry(t, newRegistryCtx(t))
	require.Error(t, err)
	require.Contains(t, err.Error(), "usage")
}

func TestPkgRegistryAddListRm(t *testing.T) {
	ctx := newRegistryCtx(t)

	require.NoError(t, runRegistry(t, ctx, "list"))
	require.Equal(t, "official\t"+OfficialDistURL+"\n", stdout(ctx))

	require.NoError(t, runRegistry(t, ctx, "add", "lab", "https://lab.example.org/"))
	require.NoError(t, runRegistry(t, ctx, "add", "acme", "https://acme.example.org/dist/"))

	// Written to registries.yml.
	cfg, err := config.Load(ctx.ConfigDir)
	require.NoError(t, err)
	require.Equal(t, "https://lab.example.org/", cfg.Registries["lab"].URL)

	require.NoError(t, runRegistry(t, ctx, "list"))
	require.Equal(t, "official\t"+OfficialDistURL+"\n"+
		"acme\thttps://acme.example.org/dist/\n"+
		"lab\thttps://lab.example.org/\n", stdout(ctx))

	require.NoError(t, runRegistry(t, ctx, "rm", "lab"))
	require.NoError(t, runRegistry(t, ctx, "list"))
	require.Equal(t, "official\t"+OfficialDistURL+"\n"+
		"acme\thttps://acme.example.org/dist/\n", stdout(ctx))

	cfg, err = config.Load(ctx.ConfigDir)
	require.NoError(t, err)
	require.NotContains(t, cfg.Registries, "lab")
}

func TestPkgRegistryAddRefused(t *testing.T) {
	ctx := newRegistryCtx(t)
	require.NoError(t, runRegistry(t, ctx, "add", "lab", "https://lab.example.org/"))

	for _, args := range [][]string{
		{"add", "lab", "https://other.example.org/"},       // duplicate
		{"add", "bad/name", "https://other.example.org/"},  // invalid name
		{"add", "official", "https://other.example.org/"},  // reserved
		{"add", "plain", "http://other.example.org/"},      // not https
		{"add", "creds", "https://u:p@other.example.org/"}, // userinfo
		{"add", "query", "https://other.example.org/?x=1"}, // query
		{"add", "nohost", "https:///dist/"},                // no host
		{"add", "lab2"},                                    // missing URL
		{"add", "a", "https://a.example.org/", "extra"},    // too many
	} {
		require.Error(t, runRegistry(t, ctx, args...), "args=%v", args)
	}

	// Nothing but the first registry was saved.
	cfg, err := config.Load(ctx.ConfigDir)
	require.NoError(t, err)
	require.Equal(t, map[string]config.RegistryConfig{
		"lab": {URL: "https://lab.example.org/"},
	}, cfg.Registries)
}

func TestPkgRegistryRmUnknown(t *testing.T) {
	ctx := newRegistryCtx(t)
	err := runRegistry(t, ctx, "rm", "nope")
	require.Error(t, err)
	require.Contains(t, err.Error(), "nope")
	require.Error(t, runRegistry(t, ctx, "rm"))
}

// A test key from the signify fixtures.
const testPubKey = "untrusted comment: other key public key\nRWRExi70uqURYN5GuknrEjksYrWkCDE3t+p++DyjElWnRTGYMOKNQYU4\n"

func TestPkgRegistryAddTrustReminder(t *testing.T) {
	ctx := newRegistryCtx(t)
	require.NoError(t, runRegistry(t, ctx, "add", "lab", "https://lab.example.org/dist/"))
	msg := stderr(ctx)
	require.Contains(t, msg, "lab")
	require.Contains(t, msg, filepath.Join(ctx.ConfigDir, "trust"))
	require.Contains(t, msg, "https://lab.example.org/dist/")

	trust := filepath.Join(ctx.ConfigDir, "trust")
	require.NoError(t, os.MkdirAll(trust, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(trust, "acme.pub"), []byte(testPubKey), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(trust, "acme.scope"), []byte("https://acme.example.org/\n"), 0o600))

	require.NoError(t, runRegistry(t, ctx, "add", "acme", "https://acme.example.org/"))
	require.Empty(t, stderr(ctx))
}

func indexServer(t *testing.T, entries ...ppkg.Integration) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(ppkg.IntegrationIndex{Version: "v1.0.0", Integrations: entries})
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/integrations-"+ppkg.PLUGIN_BUNDLE_VERSION+".json") {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func indexEntry(name string) ppkg.Integration {
	return ppkg.Integration{
		Name:    name,
		Edition: "community",
		API:     ppkg.PLUGIN_API_VERSION,
		Version: "v1.0.0",
	}
}

func TestPkgListAvailableRegistries(t *testing.T) {
	official := indexServer(t, indexEntry("s3"))
	lab := indexServer(t, indexEntry("talos"), indexEntry("s3"))
	down := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(down.Close)

	ctx := newCtx(t)
	backend, err := ppkg.NewFlatBackend(ctx.GetInner(),
		filepath.Join(ctx.CWD, "plugins"), filepath.Join(ctx.CWD, "cache"),
		&ppkg.FlatBackendOptions{})
	require.NoError(t, err)
	manager, err := ppkg.New(backend, &ppkg.Options{
		ApiURL: official.URL,
		Registries: []ppkg.Registry{
			{Name: "down", URL: down.URL},
			{Name: "lab", URL: lab.URL},
		},
	})
	require.NoError(t, err)
	ctx.SetPkgManager(manager)

	cmd := &PkgList{}
	require.NoError(t, cmd.Parse(ctx, []string{"-available"}))
	_, err = cmd.Execute(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, "s3@v1.0.0\ntalos@v1.0.0\tlab\n", stdout(ctx))
	warnings := stderr(ctx)
	require.Contains(t, warnings, "warning: registry down")
	require.Contains(t, warnings, "warning: registry lab: integration s3 shadowed by official\n")
	for _, line := range strings.Split(strings.TrimSpace(warnings), "\n") {
		require.True(t, strings.HasPrefix(line, "warning: "), "line %q", line)
	}
}

func TestPkgAddUnknownRegistry(t *testing.T) {
	ctx := newCtx(t)
	plugdir := filepath.Join(ctx.CWD, "plugins")
	backend, err := ppkg.NewFlatBackend(ctx.GetInner(), plugdir,
		filepath.Join(ctx.CWD, "cache"), &ppkg.FlatBackendOptions{})
	require.NoError(t, err)

	ptar := fmt.Sprintf("talos_v1.0.0_%s_%s.ptar", runtime.GOOS, runtime.GOARCH)
	require.NoError(t, os.MkdirAll(plugdir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(plugdir, ptar), nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(plugdir, ptar+".origin"),
		[]byte("https://gone.example.org/\n"), 0o600))

	manager, err := ppkg.New(backend, &ppkg.Options{DistURL: "https://127.0.0.1:1/"})
	require.NoError(t, err)
	ctx.SetPkgManager(manager)

	cmd := &PkgAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"-u", "talos"}))
	_, err = cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ppkg.ErrUnknownRegistry)
	require.Contains(t, err.Error(), "plakar pkg registry add <name> <url>")
	require.Equal(t, 1, strings.Count(err.Error(), "talos"), "err %q", err)
}

// "pkg add -u" without arguments warns about a package whose registry was
// removed and goes on with the next one.
func TestPkgAddUpgradeAllSkipsUnknownRegistry(t *testing.T) {
	ctx := newCtx(t)
	plugdir := filepath.Join(ctx.CWD, "plugins")
	backend, err := ppkg.NewFlatBackend(ctx.GetInner(), plugdir,
		filepath.Join(ctx.CWD, "cache"), &ppkg.FlatBackendOptions{})
	require.NoError(t, err)

	require.NoError(t, os.MkdirAll(plugdir, 0o700))
	for _, name := range []string{"talos", "vault"} {
		ptar := fmt.Sprintf("%s_v1.0.0_%s_%s.ptar", name, runtime.GOOS, runtime.GOARCH)
		require.NoError(t, os.WriteFile(filepath.Join(plugdir, ptar), nil, 0o600))
		if name == "talos" {
			require.NoError(t, os.WriteFile(filepath.Join(plugdir, ptar+".origin"),
				[]byte("https://gone.example.org/\n"), 0o600))
		}
	}

	// vault comes from the official tree, which can't be reached.
	manager, err := ppkg.New(backend, &ppkg.Options{DistURL: "https://127.0.0.1:1/"})
	require.NoError(t, err)
	ctx.SetPkgManager(manager)

	cmd := &PkgAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"-u"}))
	_, err = cmd.Execute(ctx, nil)
	require.Error(t, err)
	require.NotErrorIs(t, err, ppkg.ErrUnknownRegistry)
	require.Contains(t, err.Error(), "failed to update vault")

	warnings := stderr(ctx)
	require.True(t, strings.HasPrefix(warnings, "warning: failed to update: talos: "), "stderr %q", warnings)
	require.Contains(t, warnings, "plakar pkg registry add <name> <url>")
	require.Equal(t, 1, strings.Count(warnings, "\n"), "stderr %q", warnings)
}
