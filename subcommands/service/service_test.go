package services

import (
	"bytes"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

// newContext builds a hermetic repository + appcontext. The cookies manager
// points at a temp dir with no auth token, and PLAKAR_TOKEN is cleared, so any
// Execute that reaches getClient() fails with the "requires login" error
// before any network call to api.plakar.io is attempted.
func newContext(t *testing.T) (*repository.Repository, *appcontext.AppContext, *bytes.Buffer, *bytes.Buffer) {
	t.Setenv("PLAKAR_TOKEN", "")
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	return repo, ctx, bufOut, bufErr
}

// ---------------------------------------------------------------------------
// Parse tests
// ---------------------------------------------------------------------------

func TestServiceParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	// The bare `service` command always errors (no action specified), whether
	// or not extra args are present.
	t.Run("no action", func(t *testing.T) {
		cmd := &Service{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "no action specified")
	})
	t.Run("invalid argument", func(t *testing.T) {
		cmd := &Service{}
		err := cmd.Parse(ctx, []string{"bogus"})
		require.EqualError(t, err, "invalid argument: bogus")
	})
}

func TestServiceListParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok", func(t *testing.T) {
		cmd := &ServiceList{}
		err := cmd.Parse(ctx, []string{})
		require.NoError(t, err)
	})
	t.Run("too many args", func(t *testing.T) {
		cmd := &ServiceList{}
		err := cmd.Parse(ctx, []string{"extra"})
		require.EqualError(t, err, "invalid argument: extra")
	})
}

func TestServiceStatusParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok", func(t *testing.T) {
		cmd := &ServiceStatus{}
		err := cmd.Parse(ctx, []string{"alerting"})
		require.NoError(t, err)
		require.Equal(t, "alerting", cmd.Service)
	})
	t.Run("no arg", func(t *testing.T) {
		cmd := &ServiceStatus{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "invalid number of arguments, expected 1 but got 0")
	})
	t.Run("too many args", func(t *testing.T) {
		cmd := &ServiceStatus{}
		err := cmd.Parse(ctx, []string{"a", "b"})
		require.EqualError(t, err, "invalid number of arguments, expected 1 but got 2")
	})
}

func TestServiceEnableParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok", func(t *testing.T) {
		cmd := &ServiceEnable{}
		err := cmd.Parse(ctx, []string{"alerting"})
		require.NoError(t, err)
		require.Equal(t, "alerting", cmd.Service)
	})
	t.Run("no arg", func(t *testing.T) {
		cmd := &ServiceEnable{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "invalid number of arguments, expected 1 but got 0")
	})
	t.Run("too many args", func(t *testing.T) {
		cmd := &ServiceEnable{}
		err := cmd.Parse(ctx, []string{"a", "b"})
		require.EqualError(t, err, "invalid number of arguments, expected 1 but got 2")
	})
}

func TestServiceDisableParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok", func(t *testing.T) {
		cmd := &ServiceDisable{}
		err := cmd.Parse(ctx, []string{"alerting"})
		require.NoError(t, err)
		require.Equal(t, "alerting", cmd.Service)
	})
	t.Run("no arg", func(t *testing.T) {
		cmd := &ServiceDisable{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "invalid number of arguments, expected 1 but got 0")
	})
}

func TestServiceShowParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok with flags", func(t *testing.T) {
		cmd := &ServiceShow{}
		err := cmd.Parse(ctx, []string{"-json", "-secrets", "alerting"})
		require.NoError(t, err)
		require.Equal(t, "alerting", cmd.Service)
		require.True(t, cmd.AsJson)
		require.True(t, cmd.ShowSecrets)
	})
	t.Run("no arg", func(t *testing.T) {
		cmd := &ServiceShow{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "invalid number of arguments, expected 1 but got 0")
	})
	t.Run("too many args", func(t *testing.T) {
		cmd := &ServiceShow{}
		err := cmd.Parse(ctx, []string{"a", "b"})
		require.EqualError(t, err, "invalid number of arguments, expected 1 but got 2")
	})
}

func TestServiceAddParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok with kv", func(t *testing.T) {
		cmd := &ServiceAdd{}
		err := cmd.Parse(ctx, []string{"alerting", "email=foo@bar.io", "level=warn"})
		require.NoError(t, err)
		require.Equal(t, "alerting", cmd.Service)
		require.Equal(t, map[string]string{"email": "foo@bar.io", "level": "warn"}, cmd.Keys)
	})
	t.Run("ok empty value", func(t *testing.T) {
		cmd := &ServiceAdd{}
		err := cmd.Parse(ctx, []string{"alerting", "email="})
		require.NoError(t, err)
		require.Equal(t, map[string]string{"email": ""}, cmd.Keys)
	})
	t.Run("no service", func(t *testing.T) {
		cmd := &ServiceAdd{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "no service specified")
	})
	t.Run("invalid kv (no =)", func(t *testing.T) {
		cmd := &ServiceAdd{}
		err := cmd.Parse(ctx, []string{"alerting", "noequals"})
		require.EqualError(t, err, `invalid argument "noequals"`)
	})
	t.Run("invalid kv (empty key)", func(t *testing.T) {
		cmd := &ServiceAdd{}
		err := cmd.Parse(ctx, []string{"alerting", "=value"})
		require.EqualError(t, err, `invalid argument "=value"`)
	})
}

func TestServiceSetParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok with kv", func(t *testing.T) {
		cmd := &ServiceSet{}
		err := cmd.Parse(ctx, []string{"alerting", "email=foo@bar.io"})
		require.NoError(t, err)
		require.Equal(t, "alerting", cmd.Service)
		require.Equal(t, map[string]string{"email": "foo@bar.io"}, cmd.Keys)
	})
	t.Run("no service", func(t *testing.T) {
		cmd := &ServiceSet{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "no service specified")
	})
	t.Run("invalid kv", func(t *testing.T) {
		cmd := &ServiceSet{}
		err := cmd.Parse(ctx, []string{"alerting", "bad"})
		require.EqualError(t, err, `invalid argument "bad"`)
	})
}

func TestServiceUnsetParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok with keys", func(t *testing.T) {
		cmd := &ServiceUnset{}
		err := cmd.Parse(ctx, []string{"alerting", "email", "level"})
		require.NoError(t, err)
		require.Equal(t, "alerting", cmd.Service)
		require.Equal(t, []string{"email", "level"}, cmd.Keys)
	})
	t.Run("no service", func(t *testing.T) {
		cmd := &ServiceUnset{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "no service specified")
	})
}

func TestServiceRmParse(t *testing.T) {
	_, ctx, _, _ := newContext(t)

	t.Run("ok", func(t *testing.T) {
		cmd := &ServiceRm{}
		err := cmd.Parse(ctx, []string{"alerting"})
		require.NoError(t, err)
		require.Equal(t, "alerting", cmd.Service)
	})
	t.Run("no service", func(t *testing.T) {
		cmd := &ServiceRm{}
		err := cmd.Parse(ctx, []string{})
		require.EqualError(t, err, "no service specified")
	})
	t.Run("too many args", func(t *testing.T) {
		cmd := &ServiceRm{}
		err := cmd.Parse(ctx, []string{"a", "b"})
		require.EqualError(t, err, `invalid argument "b"`)
	})
}

// ---------------------------------------------------------------------------
// Execute tests
//
// The hardcoded SERVICE_ENDPOINT (https://api.plakar.io) cannot be overridden
// without modifying source, so the only hermetic Execute path is the
// getClient() failure: with no auth token configured, every subcommand returns
// status 1 and the "requires login" error before any network call. The
// network-backed success paths (GetServiceList, SetServiceStatus, etc.) are
// deliberately not covered here.
// ---------------------------------------------------------------------------

// With no auth-token file present and PLAKAR_TOKEN unset, GetAuthToken surfaces
// this error, which getClient propagates verbatim.
const wantLoginErr = "no auth token found: use `plakar login` first"

func TestServiceBareExecute(t *testing.T) {
	repo, ctx, _, _ := newContext(t)
	cmd := &Service{}
	status, err := cmd.Execute(ctx, repo)
	require.Equal(t, 1, status)
	require.EqualError(t, err, "no action specified")
}

func TestExecuteRequiresLogin(t *testing.T) {
	repo, ctx, _, _ := newContext(t)

	cmds := []struct {
		name string
		cmd  subExecuter
	}{
		{"list", &ServiceList{}},
		{"status", &ServiceStatus{Service: "alerting"}},
		{"enable", &ServiceEnable{Service: "alerting"}},
		{"disable", &ServiceDisable{Service: "alerting"}},
		{"show", &ServiceShow{Service: "alerting"}},
		{"add", &ServiceAdd{Service: "alerting", Keys: map[string]string{"k": "v"}}},
		{"rm", &ServiceRm{Service: "alerting"}},
		// set/unset with keys reach getClient (empty keys short-circuit to 0)
		{"set", &ServiceSet{Service: "alerting", Keys: map[string]string{"k": "v"}}},
		{"unset", &ServiceUnset{Service: "alerting", Keys: []string{"k"}}},
	}

	for _, tc := range cmds {
		t.Run(tc.name, func(t *testing.T) {
			status, err := tc.cmd.Execute(ctx, repo)
			require.Equal(t, 1, status)
			require.EqualError(t, err, wantLoginErr)
		})
	}
}

// set/unset call getClient() before the empty-keys short-circuit, so with no
// auth token they still surface the login error regardless of keys.
func TestSetUnsetNoKeysRequiresLogin(t *testing.T) {
	repo, ctx, _, _ := newContext(t)

	t.Run("set no keys", func(t *testing.T) {
		cmd := &ServiceSet{Service: "alerting", Keys: map[string]string{}}
		status, err := cmd.Execute(ctx, repo)
		require.Equal(t, 1, status)
		require.EqualError(t, err, wantLoginErr)
	})
	t.Run("unset no keys", func(t *testing.T) {
		cmd := &ServiceUnset{Service: "alerting", Keys: nil}
		status, err := cmd.Execute(ctx, repo)
		require.Equal(t, 1, status)
		require.EqualError(t, err, wantLoginErr)
	})
}

type subExecuter interface {
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
}
