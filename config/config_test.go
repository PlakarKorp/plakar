package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasRepository(t *testing.T) {
	cfg := &Config{
		Repositories: make(map[string]RepositoryConfig),
	}

	// Test non-existent repository
	require.False(t, cfg.HasRepository("test-repo"))

	// Test existing repository
	cfg.Repositories["test-repo"] = RepositoryConfig{"location": "/test/path"}
	require.True(t, cfg.HasRepository("test-repo"))
}

func TestGetRepository(t *testing.T) {
	cfg := &Config{
		Repositories: make(map[string]RepositoryConfig),
	}

	// Test direct path
	repo, err := cfg.GetRepository("/test/path")
	require.NoError(t, err)
	require.Equal(t, "/test/path", repo["location"])

	// Test non-existent repository
	_, err = cfg.GetRepository("@nonexistent")
	require.Error(t, err)

	// Test repository without location
	cfg.Repositories["test-repo"] = RepositoryConfig{"other": "value"}
	_, err = cfg.GetRepository("@test-repo")
	require.Error(t, err)

	// Test valid repository
	cfg.Repositories["test-repo"] = RepositoryConfig{"location": "/test/path"}
	repo, err = cfg.GetRepository("@test-repo")
	require.NoError(t, err)
	require.Equal(t, "/test/path", repo["location"])
}

func TestHasSource(t *testing.T) {
	cfg := &Config{
		Sources: make(map[string]SourceConfig),
	}

	// Test non-existent source
	require.False(t, cfg.HasSource("test-source"))

	// Test existing source
	cfg.Sources["test-source"] = SourceConfig{"url": "test://url"}
	require.True(t, cfg.HasSource("test-source"))
}

func TestGetSource(t *testing.T) {
	cfg := &Config{
		Sources: make(map[string]SourceConfig),
	}

	// Test non-existent source
	source, ok := cfg.GetSource("test-source")
	require.False(t, ok)
	require.Nil(t, source)

	// Test existing source
	cfg.Sources["test-source"] = SourceConfig{"url": "test://url"}
	source, ok = cfg.GetSource("test-source")
	require.True(t, ok)
	require.Equal(t, "test://url", source["url"])
}
