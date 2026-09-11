package services

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// newCoverageCtx builds a hermetic AppContext: a fresh empty cookies.Manager
// rooted in a temp dir and no PLAKAR_TOKEN, so getClient always fails with a
// login error before any network access.
func newCoverageCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	t.Setenv("PLAKAR_TOKEN", "")
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bytes.NewBuffer(nil)
	ctx.Stderr = bytes.NewBuffer(nil)
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	return ctx
}

func TestCoverageServiceLookupAndTrailingArgs(t *testing.T) {
	// Lookup must strip the matched command tokens and return the trailing args.
	cmd, _, rest := subcommands.Lookup([]string{"service", "show", "alerting", "-json"})
	require.NotNil(t, cmd)
	require.IsType(t, &ServiceShow{}, cmd)
	require.Equal(t, []string{"alerting", "-json"}, rest)

	// Bare "service" resolves to the top-level Service command with no trailing args.
	cmd, _, rest = subcommands.Lookup([]string{"service"})
	require.NotNil(t, cmd)
	require.IsType(t, &Service{}, cmd)
	require.Empty(t, rest)
}

func TestCoverageServiceParseGood(t *testing.T) {
	ctx := newCoverageCtx(t)

	// Subcommands taking exactly one <name>.
	for _, name := range []string{"status", "enable", "disable", "show", "rm"} {
		cmd, _, rest := subcommands.Lookup([]string{"service", name, "alerting"})
		require.NotNil(t, cmd, "name=%s", name)
		require.NoError(t, cmd.Parse(ctx, rest), "name=%s", name)
	}

	// list takes no args.
	require.NoError(t, (&ServiceList{}).Parse(ctx, []string{}))
}

func TestCoverageServiceParseBadNoAction(t *testing.T) {
	ctx := newCoverageCtx(t)

	// Top-level service: no action specified.
	err := (&Service{}).Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no action specified")

	// Top-level service: stray positional argument.
	err = (&Service{}).Parse(ctx, []string{"bogus"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid argument")
}

func TestCoverageServiceParseWrongArgCounts(t *testing.T) {
	ctx := newCoverageCtx(t)

	// Commands that require exactly one argument: zero args -> error.
	for _, name := range []string{"status", "enable", "disable", "show"} {
		cmd, _, rest := subcommands.Lookup([]string{"service", name})
		require.NotNil(t, cmd, "name=%s", name)
		require.Error(t, cmd.Parse(ctx, rest), "name=%s zero args", name)
	}

	// Same commands: too many args -> error.
	for _, name := range []string{"status", "enable", "disable", "show"} {
		cmd, _, rest := subcommands.Lookup([]string{"service", name, "a", "b"})
		require.NotNil(t, cmd, "name=%s", name)
		require.Error(t, cmd.Parse(ctx, rest), "name=%s two args", name)
	}

	// rm: zero args -> error; too many args -> error.
	require.Error(t, (&ServiceRm{}).Parse(ctx, []string{}))
	require.Error(t, (&ServiceRm{}).Parse(ctx, []string{"a", "b"}))

	// list: any positional arg -> error.
	require.Error(t, (&ServiceList{}).Parse(ctx, []string{"extra"}))

	// add/set/unset: zero args -> "no service specified".
	require.Error(t, (&ServiceAdd{}).Parse(ctx, []string{}))
	require.Error(t, (&ServiceSet{}).Parse(ctx, []string{}))
	require.Error(t, (&ServiceUnset{}).Parse(ctx, []string{}))
}

func TestCoverageServiceAddSetKeyValueParsing(t *testing.T) {
	ctx := newCoverageCtx(t)

	// Valid key=value pairs populate the Keys map.
	add := &ServiceAdd{}
	require.NoError(t, add.Parse(ctx, []string{"alerting", "email=foo@bar", "level=high"}))
	require.Equal(t, "alerting", add.Service)
	require.Equal(t, map[string]string{"email": "foo@bar", "level": "high"}, add.Keys)

	// Empty value is allowed (key= -> "").
	add2 := &ServiceAdd{}
	require.NoError(t, add2.Parse(ctx, []string{"alerting", "key="}))
	require.Equal(t, map[string]string{"key": ""}, add2.Keys)

	// Missing '=' is rejected.
	require.Error(t, (&ServiceAdd{}).Parse(ctx, []string{"alerting", "noequals"}))
	// Empty key (=value) is rejected.
	require.Error(t, (&ServiceAdd{}).Parse(ctx, []string{"alerting", "=value"}))

	// Same rules for set.
	set := &ServiceSet{}
	require.NoError(t, set.Parse(ctx, []string{"alerting", "a=1", "b=2"}))
	require.Equal(t, map[string]string{"a": "1", "b": "2"}, set.Keys)
	require.Error(t, (&ServiceSet{}).Parse(ctx, []string{"alerting", "noequals"}))
	require.Error(t, (&ServiceSet{}).Parse(ctx, []string{"alerting", "=value"}))

	// add/set with only a name and no pairs -> empty map, no error.
	addName := &ServiceAdd{}
	require.NoError(t, addName.Parse(ctx, []string{"alerting"}))
	require.Empty(t, addName.Keys)
}

func TestCoverageServiceUnsetKeySlice(t *testing.T) {
	ctx := newCoverageCtx(t)

	// unset populates the Keys slice from the trailing positional args.
	unset := &ServiceUnset{}
	require.NoError(t, unset.Parse(ctx, []string{"alerting", "email", "level"}))
	require.Equal(t, "alerting", unset.Service)
	require.Equal(t, []string{"email", "level"}, unset.Keys)

	// Name only -> empty key slice.
	unsetName := &ServiceUnset{}
	require.NoError(t, unsetName.Parse(ctx, []string{"alerting"}))
	require.Empty(t, unsetName.Keys)
}

func TestCoverageServiceShowParseFlags(t *testing.T) {
	ctx := newCoverageCtx(t)

	show := &ServiceShow{}
	require.NoError(t, show.Parse(ctx, []string{"-json", "-secrets", "alerting"}))
	require.True(t, show.AsJson)
	require.True(t, show.ShowSecrets)
	require.False(t, show.AsYaml)
	require.Equal(t, "alerting", show.Service)

	showYaml := &ServiceShow{}
	require.NoError(t, showYaml.Parse(ctx, []string{"-yaml", "alerting"}))
	require.True(t, showYaml.AsYaml)
	require.False(t, showYaml.AsJson)

	// show with no name -> error.
	require.Error(t, (&ServiceShow{}).Parse(ctx, []string{"-json"}))
	// show with too many names -> error.
	require.Error(t, (&ServiceShow{}).Parse(ctx, []string{"a", "b"}))
}

// TestCoverageServiceExecuteRequiresLogin drives Execute for every subcommand
// through the hermetic getClient-failure path: with no auth token, getClient
// returns a login error and Execute returns status 1 before any network call.
// The repository is a real ptesting-generated repo so the ctx is fully wired.
func TestCoverageServiceExecuteRequiresLogin(t *testing.T) {
	t.Setenv("PLAKAR_TOKEN", "")
	var out, errb bytes.Buffer
	repo, ctx := ptesting.GenerateRepository(t, &out, &errb, nil)
	// Ensure a fresh empty cookies manager (no token on disk).
	ctx.SetCookies(cookies.NewManager(t.TempDir()))

	cmds := []struct {
		name string
		cmd  subcommands.Subcommand
		args []string
	}{
		{"list", &ServiceList{}, []string{}},
		{"status", &ServiceStatus{}, []string{"alerting"}},
		{"enable", &ServiceEnable{}, []string{"alerting"}},
		{"disable", &ServiceDisable{}, []string{"alerting"}},
		{"show", &ServiceShow{}, []string{"alerting"}},
		{"add", &ServiceAdd{}, []string{"alerting", "k=v"}},
		{"set", &ServiceSet{}, []string{"alerting", "k=v"}},
		{"unset", &ServiceUnset{}, []string{"alerting", "k"}},
		{"rm", &ServiceRm{}, []string{"alerting"}},
		{"service", &Service{}, []string{}},
	}

	for _, c := range cmds {
		// Parse may legitimately error for the bare "service" command; ignore
		// that here since we only care about the Execute login path.
		_ = c.cmd.Parse(ctx, c.args)
		status, err := c.cmd.Execute(ctx, repo)
		require.Error(t, err, "subcommand=%s", c.name)
		require.Equal(t, 1, status, "subcommand=%s", c.name)
		if c.name != "service" {
			require.True(t, strings.Contains(err.Error(), "login") ||
				strings.Contains(err.Error(), "auth token"),
				"subcommand=%s expected login/auth error, got: %v", c.name, err)
		}
	}
}

// TestCoverageServiceExecuteRequiresLoginEmptyRepo runs the same login-failure
// path against a zero-value &repository.Repository{} to exercise that branch
// without a generated repo.
func TestCoverageServiceExecuteRequiresLoginEmptyRepo(t *testing.T) {
	ctx := newCoverageCtx(t)
	cmd := &ServiceShow{}
	require.NoError(t, cmd.Parse(ctx, []string{"alerting"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}
