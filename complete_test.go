package main

import (
	"os"
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// The commands that take a snapshot must be the ones that get the completion,
// and they have to exist in the tree under those names.
func TestSnapshotCompletionIsInstalled(t *testing.T) {
	_, root := newTestRoot()
	subcommands.Tree(root)
	installSnapshotCompletion(root)

	for _, path := range []string{"ls", "cat", "restore", "diag snapshot"} {
		c, _, err := root.Find(strings.Split(path, " "))
		require.NoError(t, err, "path: %s", path)
		require.NotNil(t, c.ValidArgsFunction, "path: %s", path)
	}

	// A command that takes no snapshot keeps the default behaviour.
	c, _, err := root.Find([]string{"version"})
	require.NoError(t, err)
	require.Nil(t, c.ValidArgsFunction)
}

// Every name in the list has to resolve, or the completion silently goes to a
// command nobody can reach.
func TestSnapshotArgCommandsExist(t *testing.T) {
	_, root := newTestRoot()
	subcommands.Tree(root)

	for _, path := range snapshotArgCommands {
		c, _, err := root.Find(strings.Split(path, " "))
		require.NoError(t, err, "path: %s", path)
		require.NotEqual(t, root, c, "path: %s", path)
	}
}

// Completion must answer even when there is no repository to talk to, rather
// than report the failure into the command line the user is typing.
func TestCompleteSnapshotsWithoutRepository(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PLAKAR_REPOSITORY", "/nonexistent/repository")

	out, directive := completeSnapshots(&cobra.Command{}, nil, "")
	require.Empty(t, out)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestCompletionRepositoryPathPrefersAt(t *testing.T) {
	saved := os.Args
	t.Cleanup(func() { os.Args = saved })

	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	t.Setenv("PLAKAR_REPOSITORY", "/from/env")

	os.Args = []string{"plakar", "at", "/repo", "ls"}
	require.Equal(t, "/repo", completionRepositoryPath(ctx))

	os.Args = []string{"plakar", "ls"}
	require.Equal(t, "/from/env", completionRepositoryPath(ctx))
}

// A snapshot still wants its ":path" and a directory the rest of the path, so
// the shell must not close the word behind us.
func TestCompletionKeepsTheWordOpen(t *testing.T) {
	require.False(t, isComplete([]string{"1c86e247"}))
	require.False(t, isComplete([]string{"1c86e247:/etc/"}))
	require.False(t, isComplete([]string{"1c86e247:/etc/passwd", "1c86e247:/etc/ssh/"}))

	require.True(t, isComplete([]string{"1c86e247:/etc/passwd"}))
}
