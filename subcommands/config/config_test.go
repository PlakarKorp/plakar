package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
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
	cfg, err := utils.LoadOldConfigIfExists(configPath)
	require.NoError(t, err)
	ctx := appcontext.NewAppContext()
	ctx.ConfigDir = tmpDir
	ctx.Config = cfg
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	repo := &repository.Repository{}
	args := []string{}

	subcommand := &ConfigKlosetCmd{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	expectedOutput := ``
	require.Equal(t, expectedOutput, output)

	bufOut.Reset()
	bufErr.Reset()

	args = []string{"add", "my-remote", "s3://foobar"}
	subcommandr := &ConfigSourceCmd{}
	err = subcommandr.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommandr)

	status, err = subcommandr.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	args = []string{"add", "my-repo", "fs:/tmp/foobar"}
	subcommandk := &ConfigKlosetCmd{}
	err = subcommandk.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommandk)

	status, err = subcommandk.Execute(ctx, repo)
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
	cfg, err := utils.LoadOldConfigIfExists(configPath)
	require.NoError(t, err)
	ctx := appcontext.NewAppContext()
	ctx.Config = cfg
	ctx.ConfigDir = tmpDir
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	args := []string{}
	err = source_config(ctx, args)
	require.NoError(t, err)

	args = []string{"unknown"}
	err = source_config(ctx, args)
	require.EqualError(t, err, "usage: plakar source [add|check|ls|ping|set|unset]")

	args = []string{"add", "my-remote", "invalid://my-remote"}
	err = source_config(ctx, args)
	require.NoError(t, err)

	args = []string{"add", "my-remote2", "invalid://my-remote2"}
	err = source_config(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-remote", "option=value"}
	err = source_config(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-remote2", "option2=value2"}
	err = source_config(ctx, args)
	require.NoError(t, err)

	args = []string{"unset", "my-remote2", "option2"}
	err = source_config(ctx, args)
	require.NoError(t, err)

	args = []string{"check", "my-remote"}
	err = source_config(ctx, args)
	require.EqualError(t, err, "unsupported importer protocol")
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
	cfg, err := utils.LoadOldConfigIfExists(configPath)
	require.NoError(t, err)
	ctx := appcontext.NewAppContext()
	ctx.Config = cfg
	ctx.ConfigDir = tmpDir
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	args := []string{"unknown"}
	err = cmd_kloset_config(ctx, args)
	require.EqualError(t, err, "usage: plakar kloset [add|check|ls|ping|set|unset]")

	args = []string{"add", "my-repo", "fs:/tmp/my-repo"}
	err = cmd_kloset_config(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-repo", "location=invalid://place"}
	err = cmd_kloset_config(ctx, args)
	require.NoError(t, err)

	args = []string{"default", "my-repo"}
	err = cmd_kloset_config(ctx, args)
	require.NoError(t, err)

	args = []string{"add", "my-repo2", "invalid://place2"}
	err = cmd_kloset_config(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-repo", "option=value"}
	err = cmd_kloset_config(ctx, args)
	require.NoError(t, err)

	args = []string{"set", "my-repo2", "option2=value2"}
	err = cmd_kloset_config(ctx, args)
	require.NoError(t, err)

	args = []string{"unset", "my-repo2", "option2"}
	err = cmd_kloset_config(ctx, args)
	require.NoError(t, err)

	args = []string{"check", "my-repo2"}
	err = cmd_kloset_config(ctx, args)
	require.EqualError(t, err, "backend 'invalid' does not exist")
}
