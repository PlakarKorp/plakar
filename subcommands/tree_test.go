package subcommands

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// withRegistry swaps the package registry for the duration of a test.
func withRegistry(t *testing.T, subs []subcmd) {
	t.Helper()
	saved := subcommands
	subcommands = subs
	t.Cleanup(func() { subcommands = saved })
}

func reg(use string, args ...string) subcmd {
	return subcmd{
		args:    args,
		nargs:   len(args),
		factory: func() Subcommand { return &fakeCmd{use: use} },
	}
}

func paths(root *cobra.Command) map[string]string {
	out := map[string]string{}
	var walk func(c *cobra.Command, p []string)
	walk = func(c *cobra.Command, p []string) {
		for _, k := range c.Commands() {
			path := append(append([]string{}, p...), k.Name())
			out[strings.Join(path, " ")] = k.Use
			walk(k, path)
		}
	}
	walk(root, nil)
	return out
}

func TestTreeNestsSubcommands(t *testing.T) {
	withRegistry(t, []subcmd{
		reg("diag snapshot", "diag", "snapshot"),
		reg("diag", "diag"),
		reg("version", "version"),
	})

	root := &cobra.Command{Use: "plakar"}
	Tree(root)

	got := paths(root)
	require.Contains(t, got, "diag")
	require.Contains(t, got, "diag snapshot")
	require.Contains(t, got, "version")
}

// Nothing registers a bare "token", so the tree has to invent the parent.
func TestTreeCreatesMissingParent(t *testing.T) {
	withRegistry(t, []subcmd{reg("token create", "token", "create")})

	root := &cobra.Command{Use: "plakar"}
	Tree(root)

	got := paths(root)
	require.Contains(t, got, "token")
	require.Contains(t, got, "token create")
}

// The name comes from the registration, not from Use, which does not always
// agree; the argument spec must survive.
func TestTreeNamesFromRegistration(t *testing.T) {
	withRegistry(t, []subcmd{
		reg("diag packfile", "diag", "blobsearch"),
		reg("diag blob type mac", "diag", "blob"),
		reg("cat [OPTIONS] [SNAPSHOT[:PATH]]...", "cat"),
	})

	root := &cobra.Command{Use: "plakar"}
	Tree(root)

	got := paths(root)
	require.Equal(t, "blobsearch", got["diag blobsearch"])
	require.Equal(t, "blob type mac", got["diag blob"])
	require.Equal(t, "cat [OPTIONS] [SNAPSHOT[:PATH]]...", got["cat"])
}

func TestResolve(t *testing.T) {
	withRegistry(t, []subcmd{
		reg("diag snapshot", "diag", "snapshot"),
		reg("diag", "diag"),
		reg("token create", "token", "create"),
		reg("cat", "cat"),
	})

	root := &cobra.Command{Use: "plakar"}
	Tree(root)

	// The longest path wins, without registration order arranging it.
	cmd, path, rest := Resolve(root, []string{"diag", "snapshot", "abc"})
	require.NotNil(t, cmd)
	require.Equal(t, []string{"diag", "snapshot"}, path)
	require.Equal(t, []string{"abc"}, rest)

	// Options belong to the command and are left for it to parse.
	cmd, path, rest = Resolve(root, []string{"cat", "-decompress", "snap:/f"})
	require.NotNil(t, cmd)
	require.Equal(t, []string{"cat"}, path)
	require.Equal(t, []string{"-decompress", "snap:/f"}, rest)

	// A parent nobody registered is not a command of its own.
	cmd, _, rest = Resolve(root, []string{"token"})
	require.Nil(t, cmd)
	require.Equal(t, []string{"token"}, rest)

	cmd, _, rest = Resolve(root, []string{"nosuchcmd"})
	require.Nil(t, cmd)
	require.Equal(t, []string{"nosuchcmd"}, rest)

	// An unknown word behind a command falls back to it, as before.
	cmd, path, _ = Resolve(root, []string{"diag", "nosuchsub"})
	require.NotNil(t, cmd)
	require.Equal(t, []string{"diag"}, path)
}

// The flags must bind to the value that runs, not to the tree's throwaway.
func TestResolveReturnsFreshCommand(t *testing.T) {
	withRegistry(t, []subcmd{reg("cat", "cat")})

	root := &cobra.Command{Use: "plakar"}
	Tree(root)

	a, _, _ := Resolve(root, []string{"cat"})
	b, _, _ := Resolve(root, []string{"cat"})
	require.NotNil(t, a)
	require.NotSame(t, a, b)
}

func TestArgSpec(t *testing.T) {
	require.Equal(t, " [OPTIONS] path", argSpec("backup [OPTIONS] path", 1))
	require.Equal(t, " type mac", argSpec("diag blob type mac", 2))
	require.Equal(t, "", argSpec("diag locks", 2))
	require.Equal(t, "", argSpec("version", 1))
	require.Equal(t, " [OPTIONS] <name>", argSpec("service show [OPTIONS] <name>", 2))
}
