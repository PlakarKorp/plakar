package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/stretchr/testify/require"
)

func configure(ctx *appcontext.AppContext, cmd string, args []string) error {
	subcmd := "show"
	if len(args) > 0 {
		subcmd = args[0]
		args = args[1:]
	}

	err := dispatchSubcommand(ctx, cmd, subcmd, args)
	if err != nil {
		return err
	}
	return nil
}

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

	require.Error(t, ConfigStore(ctx, nil, []string{}), "no action specified")

	require.NoError(t, ConfigStore(ctx, nil, []string{"show"}))

	output := bufOut.String()
	expectedOutput := ""
	require.Equal(t, expectedOutput, output)

	bufOut.Reset()
	bufErr.Reset()

	require.NoError(t, ConfigSource(ctx, nil, []string{"add", "my-remote", "s3://foobar"}))

	require.NoError(t, ConfigStore(ctx, nil, []string{"add", "my-repo", "fs:/tmp/foobar"}))

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
	err = configure(ctx, "source", args)
	require.NoError(t, err)

	args = []string{"unknown"}
	err = configure(ctx, "source", args)
	require.EqualError(t, err, "usage: plakar source [add|check|import|ping|rm|set|show|unset]")

	args = []string{"add", "my-remote", "invalid://my-remote"}
	err = configure(ctx, "source", args)
	require.NoError(t, err)

	args = []string{"add", "my-remote2", "invalid://my-remote2"}
	err = configure(ctx, "source", args)
	require.NoError(t, err)

	args = []string{"set", "my-remote", "option=value"}
	err = configure(ctx, "source", args)
	require.NoError(t, err)

	args = []string{"set", "my-remote2", "option2=value2"}
	err = configure(ctx, "source", args)
	require.NoError(t, err)

	args = []string{"unset", "my-remote2", "option2"}
	err = configure(ctx, "source", args)
	require.NoError(t, err)

	args = []string{"check", "my-remote"}
	err = configure(ctx, "source", args)
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
	err = configure(ctx, "store", args)
	require.EqualError(t, err, "usage: plakar store [add|check|import|ping|rm|set|show|unset]")

	args = []string{"add", "my-repo", "fs:/tmp/my-repo"}
	err = configure(ctx, "store", args)
	require.NoError(t, err)

	args = []string{"set", "my-repo", "location=invalid://place"}
	err = configure(ctx, "store", args)
	require.NoError(t, err)

	args = []string{"add", "my-repo2", "invalid://place2"}
	err = configure(ctx, "store", args)
	require.NoError(t, err)

	args = []string{"set", "my-repo", "option=value"}
	err = configure(ctx, "store", args)
	require.NoError(t, err)

	args = []string{"set", "my-repo2", "option2=value2"}
	err = configure(ctx, "store", args)
	require.NoError(t, err)

	args = []string{"unset", "my-repo2", "option2"}
	err = configure(ctx, "store", args)
	require.NoError(t, err)

	args = []string{"check", "my-repo2"}
	err = configure(ctx, "store", args)
	require.EqualError(t, err, "backend 'invalid' does not exist")
}
