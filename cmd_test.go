package main

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// newTestRoot builds the real root command with stable defaults.
func newTestRoot() (*globalOpts, *cobra.Command) {
	var opts globalOpts
	root := newRootCmd(&opts, defaults{
		configDir: "/cfg",
		cacheDir:  "/cache",
		dataDir:   "/data",
		cpuCount:  4,
	})
	return &opts, root
}

func TestNormalizeArgs(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string

		wantOut  []string
		wantRest []string
		wantRepo string
		wantAt   bool
	}{
		{
			name:     "bare subcommand",
			in:       []string{"version"},
			wantOut:  []string{},
			wantRest: []string{"version"},
		},
		{
			name:     "single dash bool flag",
			in:       []string{"-quiet", "version"},
			wantOut:  []string{"--quiet"},
			wantRest: []string{"version"},
		},
		{
			// The value of a global flag must not be mistaken for
			// the subcommand name.
			name:     "single dash flag with separate value",
			in:       []string{"-cachedir", "/tmp/c", "version"},
			wantOut:  []string{"--cachedir", "/tmp/c"},
			wantRest: []string{"version"},
		},
		{
			name:     "single dash flag with equals value",
			in:       []string{"-cachedir=/tmp/c", "version"},
			wantOut:  []string{"--cachedir=/tmp/c"},
			wantRest: []string{"version"},
		},
		{
			name:     "double dash still accepted",
			in:       []string{"--quiet", "version"},
			wantOut:  []string{"--quiet"},
			wantRest: []string{"version"},
		},
		{
			name:     "at prefix",
			in:       []string{"at", "/repo", "info"},
			wantOut:  []string{},
			wantRest: []string{"info"},
			wantRepo: "/repo",
			wantAt:   true,
		},
		{
			name:     "global flag before at prefix",
			in:       []string{"-quiet", "at", "/repo", "info"},
			wantOut:  []string{"--quiet"},
			wantRest: []string{"info"},
			wantRepo: "/repo",
			wantAt:   true,
		},
		{
			// The subcommand parses its own flags.
			name:     "subcommand flags are left alone",
			in:       []string{"at", "/repo", "create", "-plaintext"},
			wantOut:  []string{},
			wantRest: []string{"create", "-plaintext"},
			wantRepo: "/repo",
			wantAt:   true,
		},
		{
			name:     "subcommand flag colliding with a global name",
			in:       []string{"at", "/repo", "backup", "-quiet", "/data"},
			wantOut:  []string{},
			wantRest: []string{"backup", "-quiet", "/data"},
			wantRepo: "/repo",
			wantAt:   true,
		},
		{
			name:     "at with no command",
			in:       []string{"at", "/repo"},
			wantOut:  []string{},
			wantRest: nil,
			wantRepo: "/repo",
			wantAt:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, root := newTestRoot()
			out, rest, repo, hadAt, err := normalizeArgs(root, tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.wantOut, out)
			require.Equal(t, tc.wantRest, rest)
			require.Equal(t, tc.wantRepo, repo)
			require.Equal(t, tc.wantAt, hadAt)
		})
	}
}

// -h must reach us as pflag.ErrHelp so entryPoint can exit 0.
func TestGlobalFlagsHelpIsNotAnError(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		_, root := newTestRoot()
		normalized, _, _, _, err := normalizeArgs(root, []string{arg})
		require.NoError(t, err, "arg: %s", arg)

		err = root.PersistentFlags().Parse(normalized)
		require.ErrorIs(t, err, pflag.ErrHelp, "arg: %s", arg)
	}
}

func TestNormalizeArgsUnknownGlobalFlag(t *testing.T) {
	_, root := newTestRoot()
	_, _, _, _, err := normalizeArgs(root, []string{"-nosuchflag", "version"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "flag provided but not defined: -nosuchflag")
}

// Both spellings must land on the same values.
func TestGlobalFlagsParseBothSpellings(t *testing.T) {
	for _, args := range [][]string{
		{"-cachedir", "/tmp/c", "-quiet", "version"},
		{"--cachedir", "/tmp/c", "--quiet", "version"},
		{"-cachedir=/tmp/c", "-quiet", "version"},
	} {
		opts, root := newTestRoot()
		normalized, rest, _, _, err := normalizeArgs(root, args)
		require.NoError(t, err)
		require.NoError(t, root.PersistentFlags().Parse(normalized))

		require.Equal(t, "/tmp/c", opts.cacheDir, "args: %v", args)
		require.True(t, opts.quiet, "args: %v", args)
		require.Equal(t, []string{"version"}, rest, "args: %v", args)
	}
}

func TestIsCompletionRequest(t *testing.T) {
	require.True(t, isCompletionRequest([]string{cobra.ShellCompRequestCmd, "dia"}))
	require.True(t, isCompletionRequest([]string{cobra.ShellCompNoDescRequestCmd}))
	require.True(t, isCompletionRequest([]string{"completion", "bash"}))

	require.False(t, isCompletionRequest(nil))
	require.False(t, isCompletionRequest([]string{"backup", "/data"}))
	// Only the first word counts: "at <repo> completion" is a repository
	// called completion, not a request for the script.
	require.False(t, isCompletionRequest([]string{"at", "/repo", "completion"}))
}

// Completion has to name the commands, which cobra only does for the ones it
// considers runnable.
func TestCompletionListsCommands(t *testing.T) {
	_, root := newTestRoot()
	subcommands.Tree(root)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{cobra.ShellCompRequestCmd, "dia"})
	require.NoError(t, root.Execute())
	require.Contains(t, out.String(), "diag")

	out.Reset()
	root.SetArgs([]string{cobra.ShellCompRequestCmd, "diag", ""})
	require.NoError(t, root.Execute())
	require.Contains(t, out.String(), "snapshot")

	out.Reset()
	root.SetArgs([]string{cobra.ShellCompRequestCmd, "backup", "-"})
	require.NoError(t, root.Execute())
	require.Contains(t, out.String(), "--tag")
}

// Cobra resolves against the command tree, where "at" does not exist, so the
// prefix has to go before the tree sees the line.
func TestCompletionArgsDropsAtPrefix(t *testing.T) {
	require.Equal(t,
		[]string{"__complete", "ls", ""},
		completionArgs([]string{"__complete", "at", "/repo", "ls", ""}))

	require.Equal(t,
		[]string{"__complete", "ls", "1c8"},
		completionArgs([]string{"__complete", "at", "/repo", "ls", "1c8"}))

	// Without the prefix there is nothing to drop.
	require.Equal(t,
		[]string{"__complete", "ls", ""},
		completionArgs([]string{"__complete", "ls", ""}))

	// The word being completed is the repository itself: leave it, or cobra
	// is handed nothing to complete at all.
	require.Equal(t,
		[]string{"__complete", "at", ""},
		completionArgs([]string{"__complete", "at", ""}))
	require.Equal(t,
		[]string{"__complete", "at", "/tmp/re"},
		completionArgs([]string{"__complete", "at", "/tmp/re"}))
}
