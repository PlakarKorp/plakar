package services

import (
	"bytes"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
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
	cookiesManager, err := cookies.NewManager(t.TempDir())
	require.NoError(t, err)
	ctx.SetCookies(cookiesManager)
	return ctx
}

func TestServiceRegisteredFactories(t *testing.T) {
	cases := []struct {
		args []string
		typ  interface{}
	}{
		{[]string{"service", "list"}, &ServiceList{}},
		{[]string{"service", "status"}, &ServiceStatus{}},
		{[]string{"service", "enable"}, &ServiceEnable{}},
		{[]string{"service", "disable"}, &ServiceDisable{}},
		{[]string{"service", "set"}, &ServiceSet{}},
		{[]string{"service", "unset"}, &ServiceUnset{}},
		{[]string{"service", "add"}, &ServiceAdd{}},
		{[]string{"service", "rm"}, &ServiceRm{}},
		{[]string{"service", "show"}, &ServiceShow{}},
	}
	for _, c := range cases {
		cmd, _, _ := subcommands.Lookup(c.args)
		require.NotNil(t, cmd, "args=%v", c.args)
		require.IsType(t, c.typ, cmd)
	}
}

func TestServiceListParse(t *testing.T) {
	ctx := newCtx(t)
	cmd := &ServiceList{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	// extra argument is rejected
	require.Error(t, (&ServiceList{}).Parse(ctx, []string{"extra"}))
}

func TestServiceListExecuteRequiresLogin(t *testing.T) {
	// With no auth token configured, getClient fails with a "requires login"
	// error and Execute returns status 1.
	ctx := newCtx(t)
	cmd := &ServiceList{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestServiceStatusExecuteRequiresLogin(t *testing.T) {
	ctx := newCtx(t)
	cmd := &ServiceStatus{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestServiceTopLevelRequiresAction(t *testing.T) {
	// The bare "service" command (no subcommand) reports "no action specified"
	// from both Parse and Execute.
	ctx := newCtx(t)
	cmd := &Service{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no action specified")

	// A stray positional argument is rejected too.
	require.Error(t, (&Service{}).Parse(ctx, []string{"bogus"}))

	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}
