package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Repositories)
	require.NotNil(t, cfg.Sources)
	require.NotNil(t, cfg.Destinations)
	require.Empty(t, cfg.Repositories)
	require.Empty(t, cfg.Sources)
	require.Empty(t, cfg.Destinations)
	require.Equal(t, "", cfg.DefaultRepository)
}

func TestHasDestination(t *testing.T) {
	cfg := NewConfig()

	require.False(t, cfg.HasDestination("test-dest"))

	cfg.Destinations["test-dest"] = DestinationConfig{"location": "fs:///tmp"}
	require.True(t, cfg.HasDestination("test-dest"))
}

func TestGetDestination(t *testing.T) {
	cfg := NewConfig()

	// Non-existent destination
	dest, ok := cfg.GetDestination("missing")
	require.False(t, ok)
	require.Nil(t, dest)

	// Existing destination
	cfg.Destinations["test-dest"] = DestinationConfig{"location": "fs:///tmp"}
	dest, ok = cfg.GetDestination("test-dest")
	require.True(t, ok)
	require.Equal(t, "fs:///tmp", dest["location"])

	// Returned map is a copy: mutating it must not affect the stored config.
	dest["location"] = "mutated"
	again, ok := cfg.GetDestination("test-dest")
	require.True(t, ok)
	require.Equal(t, "fs:///tmp", again["location"])
}

func TestGetDestinationWithRootOverride(t *testing.T) {
	cfg := NewConfig()
	cfg.Destinations["d"] = DestinationConfig{"location": "/base/path"}

	// Relative root override is joined onto the local path.
	dest, ok := cfg.GetDestination("d:sub/dir")
	require.True(t, ok)
	require.Equal(t, "/base/path/sub/dir", dest["location"])

	// Absolute root override replaces the local path.
	dest, ok = cfg.GetDestination("d:/abs/override")
	require.True(t, ok)
	require.Equal(t, "/abs/override", dest["location"])
}

func TestGetSourceWithRootOverride(t *testing.T) {
	cfg := NewConfig()
	cfg.Sources["s"] = SourceConfig{"location": "/base/path"}

	dest, ok := cfg.GetSource("s:sub")
	require.True(t, ok)
	require.Equal(t, "/base/path/sub", dest["location"])

	// Returned map is a copy.
	dest["location"] = "mutated"
	again, ok := cfg.GetSource("s")
	require.True(t, ok)
	require.Equal(t, "/base/path", again["location"])
}

func TestGetRepositoryWithRootOverride(t *testing.T) {
	cfg := NewConfig()

	// Local path repository, relative override is joined.
	cfg.Repositories["local"] = RepositoryConfig{"location": "/base/repo"}
	repo, err := cfg.GetRepository("@local:sub")
	require.NoError(t, err)
	require.Equal(t, "/base/repo/sub", repo["location"])

	// Absolute override replaces the path.
	repo, err = cfg.GetRepository("@local:/elsewhere")
	require.NoError(t, err)
	require.Equal(t, "/elsewhere", repo["location"])

	// URL-style location: relative override is joined onto the URL path.
	cfg.Repositories["remote"] = RepositoryConfig{"location": "s3://bucket/prefix"}
	repo, err = cfg.GetRepository("@remote:more")
	require.NoError(t, err)
	require.Equal(t, "s3://bucket/prefix/more", repo["location"])

	// URL-style location: absolute override sets the URL path.
	repo, err = cfg.GetRepository("@remote:/abs")
	require.NoError(t, err)
	require.Equal(t, "s3://bucket/abs", repo["location"])
}
