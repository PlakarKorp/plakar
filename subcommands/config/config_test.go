package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/appcontext"
	"github.com/PlakarKorp/kloset/config"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/stretchr/testify/require"
)

func TestConfigEmpty(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	// init temporary directories
	tmpDir, err := os.MkdirTemp("", "plakar-config-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	configPath := filepath.Join(tmpDir, "config.yaml")
	cfg, err := config.LoadOrCreate(configPath)
	require.NoError(t, err)
	ctx := appcontext.NewAppContext()
	ctx.SetConfig(cfg)
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = tmpDir
	repo := &repository.Repository{}
	args := []string{}

	subcommand := &ConfigCmd{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	expectedOutput := `default-repo: ""
repositories: {}
remotes: {}
`
	require.Equal(t, expectedOutput, output)

	bufOut.Reset()
	bufErr.Reset()

	args = []string{"unknown"}
	subcommand = &ConfigCmd{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.ErrorContains(t, err, "unknown subcommand unknown")
	require.Equal(t, 1, status)

	args = []string{"remote", "create", "my-remote"}
	subcommand = &ConfigCmd{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	args = []string{"repository", "create", "my-repo"}
	subcommand = &ConfigCmd{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output = bufOut.String()
	expectedOutput = ``
	require.Equal(t, expectedOutput, output)
}

func TestCmdRemote(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	// init temporary directories
	tmpDir, err := os.MkdirTemp("", "plakar-config-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	configPath := filepath.Join(tmpDir, "config.yaml")
	cfg, err := config.LoadOrCreate(configPath)
	require.NoError(t, err)
	ctx := appcontext.NewAppContext()
	ctx.SetConfig(cfg)
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	args := []string{}
	err = cmd_remote(ctx, args)
	require.EqualError(t, err, "usage: plakar config remote [create | set | unset | validate]")

	args = []string{"unknown"}
	err = cmd_remote(ctx, args)
	require.EqualError(t, err, "usage: plakar config remote [create | set | unset | validate]")

	args = []string{"create", "my-remote"}
	err = cmd_remote(ctx, args)
	require.NoError(t, err)

	args = []string{"create", "my-remote2"}
	err = cmd_remote(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-remote", "option", "value"}
	err = cmd_remote(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-remote2", "option2", "value2"}
	err = cmd_remote(ctx, args)
	require.NoError(t, err)

	args = []string{"unset", "my-remote2", "option2"}
	err = cmd_remote(ctx, args)
	require.NoError(t, err)

	args = []string{"validate", "my-remote2"}
	err = cmd_remote(ctx, args)
	require.EqualError(t, err, "validation not implemented")

	ctx.RenderConfig(ctx.Stdout)

	output := bufOut.String()
	expectedOutput := `default-repo: ""
repositories: {}
remotes:
    my-remote:
        option: value
    my-remote2: {}
`
	require.Equal(t, expectedOutput, output)
}

func TestCmdRepository(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	// init temporary directories
	tmpDir, err := os.MkdirTemp("", "plakar-config-test")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	configPath := filepath.Join(tmpDir, "config.yaml")
	cfg, err := config.LoadOrCreate(configPath)
	require.NoError(t, err)
	ctx := appcontext.NewAppContext()
	ctx.SetConfig(cfg)
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	args := []string{"unknown"}
	err = cmd_repository(ctx, args)
	require.EqualError(t, err, "usage: plakar config repository [create | default | set | unset | validate]")

	args = []string{"create", "my-repo"}
	err = cmd_repository(ctx, args)
	require.NoError(t, err)

	args = []string{"default", "my-repo"}
	err = cmd_repository(ctx, args)
	require.EqualError(t, err, "repository \"my-repo\" doesn't have a location set")

	args = []string{"set", "my-repo", "location", "invalid://place"}
	err = cmd_repository(ctx, args)
	require.NoError(t, err)

	args = []string{"default", "my-repo"}
	err = cmd_repository(ctx, args)
	require.NoError(t, err)

	args = []string{"create", "my-repo2"}
	err = cmd_repository(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-repo", "option", "value"}
	err = cmd_repository(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-repo2", "option2", "value2"}
	err = cmd_repository(ctx, args)
	require.NoError(t, err)

	args = []string{"unset", "my-repo2", "option2"}
	err = cmd_repository(ctx, args)
	require.NoError(t, err)

	args = []string{"validate", "my-repo2"}
	err = cmd_repository(ctx, args)
	require.EqualError(t, err, "validation not implemented")

	ctx.RenderConfig(ctx.Stdout)

	output := bufOut.String()
	expectedOutput := `default-repo: my-repo
repositories:
    my-repo:
        location: invalid://place
        option: value
    my-repo2: {}
remotes: {}
`
	require.Equal(t, expectedOutput, output)
}
