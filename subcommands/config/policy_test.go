package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDispatchPolicy covers the actions of dispatchPolicy over a policy that
// already exists, plus the argument and lookup errors of each action.
func TestDispatchPolicy(t *testing.T) {
	t.Run("unknown-action", func(t *testing.T) {
		ctx, _, _ := newCtx(t)

		err := dispatchPolicy(ctx, "policy", "bogus", nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "usage:")
	})

	t.Run("add/ok", func(t *testing.T) {
		ctx, _, _ := newCtx(t)
		require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"daily", "tags=auto"}))
	})

	t.Run("add/no-args", func(t *testing.T) {
		ctx, _, _ := newCtx(t)
		require.Error(t, dispatchPolicy(ctx, "policy", "add", []string{}))
	})

	t.Run("add/duplicate", func(t *testing.T) {
		ctx, _, _ := newCtx(t, withPolicy("daily", "days=7"))
		require.Error(t, dispatchPolicy(ctx, "policy", "add", []string{"daily"}))
	})

	t.Run("add/malformed-kv", func(t *testing.T) {
		ctx, _, _ := newCtx(t)
		require.Error(t, dispatchPolicy(ctx, "policy", "add", []string{"weekly", "bad"}))
	})

	t.Run("add/bad-value", func(t *testing.T) {
		ctx, _, _ := newCtx(t)

		// "minutes" expects a non-negative int, so config.Set fails.
		err := dispatchPolicy(ctx, "policy", "add", []string{"p", "minutes=notanint"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to set key")
	})

	t.Run("set/ok", func(t *testing.T) {
		ctx, _, _ := newCtx(t, withPolicy("daily", "days=7"))
		require.NoError(t, dispatchPolicy(ctx, "policy", "set", []string{"daily", "tags=auto,nightly"}))
	})

	t.Run("set/unknown-name", func(t *testing.T) {
		ctx, _, _ := newCtx(t)
		require.Error(t, dispatchPolicy(ctx, "policy", "set", []string{"ghost", "k=v"}))
	})

	t.Run("set/too-few-args", func(t *testing.T) {
		ctx, _, _ := newCtx(t, withPolicy("daily", "days=7"))
		require.Error(t, dispatchPolicy(ctx, "policy", "set", []string{"daily"}))
	})

	t.Run("set/malformed-kv", func(t *testing.T) {
		ctx, _, _ := newCtx(t, withPolicy("daily", "days=7"))
		require.Error(t, dispatchPolicy(ctx, "policy", "set", []string{"daily", "bad"}))
	})

	t.Run("set/bad-value", func(t *testing.T) {
		ctx, _, _ := newCtx(t, withPolicy("daily", "days=7"))

		err := dispatchPolicy(ctx, "policy", "set", []string{"daily", "minutes=notanint"})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to set key")
	})

	t.Run("show/all", func(t *testing.T) {
		ctx, bufOut, _ := newCtx(t, withPolicy("daily", "days=7"))

		require.NoError(t, dispatchPolicy(ctx, "policy", "show", nil))
		require.Contains(t, bufOut.String(), "daily")
	})

	t.Run("show/yaml", func(t *testing.T) {
		ctx, bufOut, _ := newCtx(t, withPolicy("daily", "days=7"))

		require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"daily"}))
		require.Contains(t, bufOut.String(), "daily")
	})

	t.Run("show/json", func(t *testing.T) {
		ctx, bufOut, _ := newCtx(t, withPolicy("daily", "days=7"))

		require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"-json", "daily"}))
		require.True(t, strings.HasPrefix(strings.TrimSpace(bufOut.String()), "{"))
		require.Contains(t, bufOut.String(), "daily")
	})

	t.Run("unset/ok", func(t *testing.T) {
		ctx, _, _ := newCtx(t, withPolicy("daily", "tags=auto,nightly"))
		require.NoError(t, dispatchPolicy(ctx, "policy", "unset", []string{"daily", "tags"}))
	})

	t.Run("unset/too-few-args", func(t *testing.T) {
		ctx, _, _ := newCtx(t, withPolicy("daily", "days=7"))
		require.Error(t, dispatchPolicy(ctx, "policy", "unset", []string{"daily"}))
	})

	t.Run("unset/unknown-name", func(t *testing.T) {
		ctx, _, _ := newCtx(t)
		require.Error(t, dispatchPolicy(ctx, "policy", "unset", []string{"ghost", "tags"}))
	})

	t.Run("rm/ok", func(t *testing.T) {
		ctx, _, _ := newCtx(t, withPolicy("daily", "days=7"))
		require.NoError(t, dispatchPolicy(ctx, "policy", "rm", []string{"daily"}))
	})

	t.Run("rm/unknown-name", func(t *testing.T) {
		ctx, _, _ := newCtx(t)
		require.Error(t, dispatchPolicy(ctx, "policy", "rm", []string{"ghost"}))
	})

	t.Run("rm/no-args", func(t *testing.T) {
		ctx, _, _ := newCtx(t)
		require.Error(t, dispatchPolicy(ctx, "policy", "rm", []string{}))
	})
}

// TestDispatchPolicyLoadError needs a context whose policies.yml is already
// malformed on disk.
func TestDispatchPolicyLoadError(t *testing.T) {
	ctx, _, _ := newCtx(t)

	require.NoError(t, os.WriteFile(filepath.Join(ctx.ConfigDir, "policies.yml"),
		[]byte("this: : : not valid\n  - broken"), 0644))

	err := dispatchPolicy(ctx, "policy", "show", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load config file")
}
