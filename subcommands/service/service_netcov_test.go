package services

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/stretchr/testify/require"
)

// netcovCtx builds a hermetic AppContext whose cookie jar already holds an auth
// token, and points PLAKAR_API_URL at the supplied httptest server. With both in
// place getClient() succeeds and Execute proceeds to the network call against the
// local mock instead of api.plakar.io.
//
// These tests use t.Setenv (process-global), so they must NOT call t.Parallel().
func netcovCtx(t *testing.T, srvURL string) *appcontext.AppContext {
	t.Helper()
	t.Setenv("PLAKAR_TOKEN", "")
	t.Setenv("PLAKAR_API_URL", srvURL)
	ctx := appcontext.NewAppContext()
	ctx.Client = "plakar-test"
	ctx.OperatingSystem = "testos"
	ctx.Architecture = "testarch"
	ctx.Stdout = bytes.NewBuffer(nil)
	ctx.Stderr = bytes.NewBuffer(nil)
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	require.NoError(t, ctx.GetCookies().PutAuthToken("netcov-token"))
	return ctx
}

func netcovStdout(ctx *appcontext.AppContext) string {
	return ctx.Stdout.(*bytes.Buffer).String()
}

// TestNetcovServiceListExecuteSuccess drives ServiceList.Execute end to end: the
// mock returns a two-element service list and each name is printed to stdout.
func TestNetcovServiceListExecuteSuccess(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `[{"name":"alerting"},{"name":"backup"}]`)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceList{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := netcovStdout(ctx)
	require.Contains(t, out, "alerting")
	require.Contains(t, out, "backup")
	require.Equal(t, "/v1/account/services", gotPath)
	require.Equal(t, "Bearer netcov-token", gotAuth)
}

// TestNetcovServiceListExecuteServerError covers the Execute error branch when
// the network call fails (non-200): status 1 and a non-nil error.
func TestNetcovServiceListExecuteServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceList{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestNetcovServiceStatusExecuteEnabled drives ServiceStatus.Execute when the
// service reports enabled=true.
func TestNetcovServiceStatusExecuteEnabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/account/services/alerting", r.URL.Path)
		_, _ = io.WriteString(w, `{"enabled":true}`)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceStatus{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, netcovStdout(ctx), "status: enabled")
}

// TestNetcovServiceStatusExecuteDisabled drives the enabled=false branch.
func TestNetcovServiceStatusExecuteDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"enabled":false}`)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceStatus{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, netcovStdout(ctx), "status: disabled")
}

// TestNetcovServiceStatusExecuteError covers the failing network branch.
func TestNetcovServiceStatusExecuteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusForbidden)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceStatus{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestNetcovServiceEnableExecuteSuccess drives ServiceEnable.Execute: a PUT with
// {"enabled":true} and "enabled" printed on success.
func TestNetcovServiceEnableExecuteSuccess(t *testing.T) {
	var gotMethod string
	var gotBody struct {
		Enabled bool `json:"enabled"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceEnable{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "PUT", gotMethod)
	require.True(t, gotBody.Enabled)
	require.Contains(t, netcovStdout(ctx), "enabled")
}

func TestNetcovServiceEnableExecuteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceEnable{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestNetcovServiceDisableExecuteSuccess drives ServiceDisable.Execute: a PUT
// with {"enabled":false} and "disabled" printed on success.
func TestNetcovServiceDisableExecuteSuccess(t *testing.T) {
	var gotBody struct {
		Enabled bool `json:"enabled"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceDisable{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.False(t, gotBody.Enabled)
	require.Contains(t, netcovStdout(ctx), "disabled")
}

func TestNetcovServiceDisableExecuteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceDisable{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestNetcovServiceShowExecuteYAML drives ServiceShow.Execute (default YAML) and
// asserts the rendered configuration contains the service name and a key.
func TestNetcovServiceShowExecuteYAML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/account/services/alerting/configuration", r.URL.Path)
		_, _ = io.WriteString(w, `{"webhook":"https://example.test"}`)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceShow{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	out := netcovStdout(ctx)
	require.Contains(t, out, "alerting")
	require.Contains(t, out, "webhook")
	require.Contains(t, out, "https://example.test")
}

// TestNetcovServiceShowExecuteJSON drives the -json output branch.
func TestNetcovServiceShowExecuteJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"webhook":"https://example.test"}`)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceShow{}
	require.NoError(t, cmd.Parse(ctx, []string{"-json", "alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := netcovStdout(ctx)
	require.True(t, strings.HasPrefix(strings.TrimSpace(out), "{"), "expected JSON output, got %q", out)
	var decoded map[string]map[string]string
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Equal(t, "https://example.test", decoded["alerting"]["webhook"])
}

func TestNetcovServiceShowExecuteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusForbidden)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceShow{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestNetcovServiceAddExecuteSuccess drives ServiceAdd.Execute, which first
// validates+PUTs the configuration (requiring a services-list lookup) and then
// PUTs the enabled status. The mock answers all three requests.
func TestNetcovServiceAddExecuteSuccess(t *testing.T) {
	var gotConfig map[string]string
	var statusBody struct {
		Enabled bool `json:"enabled"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/account/services":
			_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
		case "/v1/account/services/alerting/configuration":
			require.Equal(t, "PUT", r.Method)
			_ = json.NewDecoder(r.Body).Decode(&gotConfig)
			w.WriteHeader(http.StatusOK)
		case "/v1/account/services/alerting":
			require.Equal(t, "PUT", r.Method)
			_ = json.NewDecoder(r.Body).Decode(&statusBody)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting", "webhook=https://x.test"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "https://x.test", gotConfig["webhook"])
	require.True(t, statusBody.Enabled, "add must enable the service")
}

// TestNetcovServiceAddExecuteConfigError covers the branch where the
// configuration write fails (the services list resolves, but the config PUT 500s).
func TestNetcovServiceAddExecuteConfigError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/account/services" {
			_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
			return
		}
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceAdd{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting", "webhook=https://x.test"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestNetcovServiceSetExecuteSuccess drives ServiceSet.Execute: it GETs the
// current configuration, merges the new keys, and PUTs the result.
func TestNetcovServiceSetExecuteSuccess(t *testing.T) {
	var putConfig map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/account/services":
			_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
		case r.URL.Path == "/v1/account/services/alerting/configuration" && r.Method == "GET":
			_, _ = io.WriteString(w, `{"existing":"keep"}`)
		case r.URL.Path == "/v1/account/services/alerting/configuration" && r.Method == "PUT":
			_ = json.NewDecoder(r.Body).Decode(&putConfig)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceSet{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting", "newkey=newval"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "keep", putConfig["existing"], "existing keys must be preserved")
	require.Equal(t, "newval", putConfig["newkey"], "new key must be merged in")
}

// TestNetcovServiceSetExecuteNoKeys covers the short-circuit branch: with no
// key=value pairs Execute returns 0 immediately without touching the network.
func TestNetcovServiceSetExecuteNoKeys(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceSet{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.False(t, hit, "no keys must short-circuit before any network call")
}

// TestNetcovServiceSetExecuteGetError covers the branch where fetching the
// current configuration fails.
func TestNetcovServiceSetExecuteGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusForbidden)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceSet{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting", "k=v"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestNetcovServiceUnsetExecuteSuccess drives ServiceUnset.Execute: it GETs the
// configuration, deletes the requested keys, and PUTs the result.
func TestNetcovServiceUnsetExecuteSuccess(t *testing.T) {
	var putConfig map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/account/services":
			_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
		case r.URL.Path == "/v1/account/services/alerting/configuration" && r.Method == "GET":
			_, _ = io.WriteString(w, `{"keep":"yes","drop":"no"}`)
		case r.URL.Path == "/v1/account/services/alerting/configuration" && r.Method == "PUT":
			_ = json.NewDecoder(r.Body).Decode(&putConfig)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceUnset{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting", "drop"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "yes", putConfig["keep"], "untouched key must remain")
	_, dropped := putConfig["drop"]
	require.False(t, dropped, "unset key must be removed")
}

// TestNetcovServiceUnsetExecuteNoKeys covers the short-circuit branch.
func TestNetcovServiceUnsetExecuteNoKeys(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceUnset{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.False(t, hit, "no keys must short-circuit before any network call")
}

// TestNetcovServiceUnsetExecuteGetError covers the failing-GET branch.
func TestNetcovServiceUnsetExecuteGetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusForbidden)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceUnset{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting", "drop"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// TestNetcovServiceRmExecuteSuccess drives ServiceRm.Execute: it disables the
// service (PUT enabled=false) and then writes an empty configuration.
func TestNetcovServiceRmExecuteSuccess(t *testing.T) {
	var statusBody struct {
		Enabled bool `json:"enabled"`
	}
	var emptiedConfig map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/account/services":
			_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
		case "/v1/account/services/alerting":
			_ = json.NewDecoder(r.Body).Decode(&statusBody)
			w.WriteHeader(http.StatusNoContent)
		case "/v1/account/services/alerting/configuration":
			_ = json.NewDecoder(r.Body).Decode(&emptiedConfig)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceRm{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.False(t, statusBody.Enabled, "rm must disable the service")
	require.Empty(t, emptiedConfig, "rm must write an empty configuration")
}

// TestNetcovServiceRmExecuteStatusError covers the branch where disabling fails.
func TestNetcovServiceRmExecuteStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := netcovCtx(t, srv.URL)
	cmd := &ServiceRm{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}
