package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	require.NotNil(t, c)
	require.NotNil(t, c.Repositories)
	require.NotNil(t, c.Sources)
	require.NotNil(t, c.Destinations)
	require.Empty(t, c.DefaultRepository)
	require.Empty(t, c.Repositories)
}

func TestHasDestination(t *testing.T) {
	c := NewConfig()
	require.False(t, c.HasDestination("missing"))

	c.Destinations["dst"] = map[string]string{"location": "/tmp/out"}
	require.True(t, c.HasDestination("dst"))
}

func TestGetDestination(t *testing.T) {
	c := NewConfig()

	// missing
	got, err := c.GetDestination("@missing")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, got)

	// present
	c.Destinations["dst"] = map[string]string{"location": "/tmp/out", "extra": "yes"}
	got, err = c.GetDestination("@dst")
	require.NoError(t, err)
	require.Equal(t, "/tmp/out", got["location"])
	require.Equal(t, "yes", got["extra"])

	// returned map is a copy, not the underlying storage
	got["location"] = "/mutated"
	require.Equal(t, "/tmp/out", c.Destinations["dst"]["location"])
}

func TestResolveRootOverride(t *testing.T) {
	cases := []struct {
		input    string
		wantName string
		wantRoot string
	}{
		{"plain", "plain", ""},
		{"name:/abs/path", "name", "/abs/path"},
		{"name:rel/path", "name", "rel/path"},
		{"name:", "name", ""}, // trailing colon -> empty override
		{":/abs", "", "/abs"}, // leading colon -> empty name
		{"a:b:c", "a", "b:c"}, // only the first colon splits
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			gotName, gotRoot := resolveRootOverride(c.input)
			require.Equal(t, c.wantName, gotName)
			require.Equal(t, c.wantRoot, gotRoot)
		})
	}
}

func TestApplyRootOverride_EmptyOverrideReturnsLocation(t *testing.T) {
	got, err := applyRootOverride("s3://bucket/key", "")
	require.NoError(t, err)
	require.Equal(t, "s3://bucket/key", got)
}

func TestApplyRootOverride_LocalPath_AbsoluteOverrideReplaces(t *testing.T) {
	// localPath=true (starts with /), absolute override -> override wins
	got, err := applyRootOverride("/var/backups", "/other/root")
	require.NoError(t, err)
	require.Equal(t, "/other/root", got)
}

func TestApplyRootOverride_LocalPath_RelativeOverrideJoins(t *testing.T) {
	// localPath=true, relative override -> joined onto the base
	got, err := applyRootOverride("/var/backups", "sub/dir")
	require.NoError(t, err)
	require.Equal(t, "/var/backups/sub/dir", got)
}

func TestApplyRootOverride_URL_AbsoluteOverrideReplacesPath(t *testing.T) {
	// non-local (URL), absolute override -> replaces the URL's path entirely
	got, err := applyRootOverride("s3://bucket/old/path", "/new/path")
	require.NoError(t, err)
	require.Equal(t, "s3://bucket/new/path", got)
}

func TestApplyRootOverride_URL_RelativeOverrideJoinsPath(t *testing.T) {
	// non-local (URL), relative override -> joined onto the URL's existing path
	got, err := applyRootOverride("s3://bucket/old", "sub")
	require.NoError(t, err)
	require.Equal(t, "s3://bucket/old/sub", got)
}

func TestApplyRootOverride_URL_BadLocationReturnsError(t *testing.T) {
	// A malformed URL with a control character causes url.Parse to fail.
	// Only triggered on the non-local-path branch (no leading "/").
	_, err := applyRootOverride("ht\x00tp://x", "/something")
	require.Error(t, err)
}

func TestGetRepository_RootOverrideAppliedToLocalLocation(t *testing.T) {
	c := NewConfig()
	c.Repositories["repo"] = map[string]string{"location": "/var/backups"}

	got, err := c.GetRepository("@repo:/elsewhere")
	require.NoError(t, err)
	require.Equal(t, "/elsewhere", got["location"])

	got, err = c.GetRepository("@repo:sub/dir")
	require.NoError(t, err)
	require.Equal(t, "/var/backups/sub/dir", got["location"])
}

func TestGetRepository_RootOverrideAppliedToURL(t *testing.T) {
	c := NewConfig()
	c.Repositories["repo"] = map[string]string{"location": "s3://bucket/base"}

	got, err := c.GetRepository("@repo:sub")
	require.NoError(t, err)
	require.Equal(t, "s3://bucket/base/sub", got["location"])
}

func TestGetRepository_RootOverridePropagatesURLParseError(t *testing.T) {
	c := NewConfig()
	c.Repositories["repo"] = map[string]string{"location": "ht\x00tp://x"}

	_, err := c.GetRepository("@repo:/x")
	require.Error(t, err)
}

func TestGetRepository_DirectPathPassesThrough(t *testing.T) {
	c := NewConfig()
	// Anything not starting with "@" is treated as a direct path; no lookup,
	// just wrap in a map.
	got, err := c.GetRepository("/literal/path")
	require.NoError(t, err)
	require.Equal(t, "/literal/path", got["location"])
}

func TestGetSource_RootOverrideAppliedToLocalLocation(t *testing.T) {
	c := NewConfig()
	c.Sources["src"] = map[string]string{"location": "/data"}

	got, err := c.GetSource("@src:/elsewhere")
	require.NoError(t, err)
	require.Equal(t, "/elsewhere", got["location"])
}

func TestGetSource_RootOverrideReturnsFalseOnURLParseError(t *testing.T) {
	c := NewConfig()
	// GetSource swallows the applyRootOverride error and returns ok=false.
	c.Sources["src"] = map[string]string{"location": "ht\x00tp://x"}

	_, err := c.GetSource("@src:/x")
	require.Error(t, err)
}

func TestGetDestination_RootOverrideAppliedToLocalLocation(t *testing.T) {
	c := NewConfig()
	c.Destinations["dst"] = map[string]string{"location": "/out"}

	got, err := c.GetDestination("@dst:sub")
	require.NoError(t, err)
	require.Equal(t, "/out/sub", got["location"])
}

func TestGetDestination_RootOverrideReturnsFalseOnURLParseError(t *testing.T) {
	c := NewConfig()
	c.Destinations["dst"] = map[string]string{"location": "ht\x00tp://x"}

	_, err := c.GetDestination("@dst:/x")
	require.Error(t, err)
}

func TestGetRepository_ResolveRootOverrideStripsAtPrefix(t *testing.T) {
	// Sanity: the "@" prefix is on the *full* token, then the colon split
	// happens on what's left. So "@name:override" -> name lookup is "name".
	c := NewConfig()
	c.Repositories["repo"] = map[string]string{"location": "/x"}

	got, err := c.GetRepository("@repo:sub")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(got["location"], "/x/sub"))
}
