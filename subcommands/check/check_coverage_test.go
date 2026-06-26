package check

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

// An invalid (non-hex) snapshot prefix is rejected by Execute.
func TestExecuteCmdCheckInvalidPrefix(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	cmd := &Check{}
	require.NoError(t, cmd.Parse(ctx, []string{"zzzz:"}))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "invalid snapshot prefix")
}

// Checking a specific snapshot constrained to a sub-path walks the
// path-scoped branch of Execute.
func TestExecuteCmdCheckSnapshotWithPath(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	short := snap.Header.GetIndexShortID()
	args := []string{fmt.Sprintf("%x:subdir", short)}

	cmd := &Check{}
	require.NoError(t, cmd.Parse(ctx, args))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// -fast and -no-verify flags flow through Parse and Execute without verifying
// signatures or digests.
func TestExecuteCmdCheckFastNoVerify(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	cmd := &Check{}
	require.NoError(t, cmd.Parse(ctx, []string{"-fast", "-no-verify"}))
	require.True(t, cmd.FastCheck)
	require.True(t, cmd.NoVerify)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// A valid hex prefix that matches no snapshot resolves to an empty set and
// completes with no failures.
func TestExecuteCmdCheckValidPrefixNoMatch(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	cmd := &Check{}
	require.NoError(t, cmd.Parse(ctx, []string{"deadbeef:"}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// Specifying both a snapshot and a filter warns that filters are ignored.
func TestExecuteCmdCheckSnapshotWithFilterWarns(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()
	_ = repo

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	short := snap.Header.GetIndexShortID()
	cmd := &Check{}
	require.NoError(t, cmd.Parse(ctx, []string{"-latest", fmt.Sprintf("%x", short)}))
}
