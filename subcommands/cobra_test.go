package subcommands

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func testFlags() *pflag.FlagSet {
	f := pflag.NewFlagSet("test", pflag.ContinueOnError)
	f.Bool("plaintext", false, "")
	f.Bool("quiet", false, "")
	f.String("name", "", "")
	// One-letter long options, the way ptar and pkg declare -o and -u:
	// with StringVar and friends, not the shorthand-taking VarP.
	f.String("o", "", "")
	f.Bool("u", false, "")
	return f
}

func TestSingleDash(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "bool flag",
			in:   []string{"-plaintext"},
			want: []string{"--plaintext"},
		},
		{
			name: "flag with separate value",
			in:   []string{"-name", "snap"},
			want: []string{"--name", "snap"},
		},
		{
			name: "flag with equals value",
			in:   []string{"-name=snap"},
			want: []string{"--name=snap"},
		},
		{
			name: "double dash spelling untouched",
			in:   []string{"--plaintext"},
			want: []string{"--plaintext"},
		},
		{
			// -o is a long option, not a shorthand.
			name: "one letter long option with separate value",
			in:   []string{"-o", "out.txt"},
			want: []string{"--o", "out.txt"},
		},
		{
			name: "one letter long option with equals value",
			in:   []string{"-o=out.txt"},
			want: []string{"--o=out.txt"},
		},
		{
			name: "one letter long boolean option",
			in:   []string{"-u"},
			want: []string{"--u"},
		},
		{
			// Past the first positional everything is data.
			name: "stops at the first positional",
			in:   []string{"-quiet", "/data", "-notaflag"},
			want: []string{"--quiet", "/data", "-notaflag"},
		},
		{
			name: "value that looks like a flag",
			in:   []string{"-name", "-weird", "/data"},
			want: []string{"--name", "-weird", "/data"},
		},
		{
			name: "everything after a bare dashdash",
			in:   []string{"-quiet", "--", "-notaflag"},
			want: []string{"--quiet", "--", "-notaflag"},
		},
		{
			name: "no arguments",
			in:   []string{},
			want: []string{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SingleDash(testFlags(), tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// Rewriting is only half the job: the result has to survive pflag, which is
// what the one-letter options got wrong.
func TestSingleDashRoundTripsThroughPflag(t *testing.T) {
	for _, tc := range []struct {
		args []string
		flag string
		want string
	}{
		{args: []string{"-o", "out.txt"}, flag: "o", want: "out.txt"},
		{args: []string{"-o=out.txt"}, flag: "o", want: "out.txt"},
		{args: []string{"-u"}, flag: "u", want: "true"},
		{args: []string{"-name", "snap"}, flag: "name", want: "snap"},
		{args: []string{"-plaintext"}, flag: "plaintext", want: "true"},
	} {
		flags := testFlags()
		rewritten, err := SingleDash(flags, tc.args)
		require.NoError(t, err, "args: %v", tc.args)
		require.NoError(t, flags.Parse(rewritten), "args: %v", tc.args)
		require.Equal(t, tc.want, flags.Lookup(tc.flag).Value.String(),
			"args: %v", tc.args)
	}
}

func TestSingleDashUnknownFlag(t *testing.T) {
	_, err := SingleDash(testFlags(), []string{"-nope"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "flag provided but not defined: -nope")
}

// A dash-leading path after a positional must be left alone.
func TestSingleDashLeavesTrailingDataAlone(t *testing.T) {
	got, err := SingleDash(testFlags(), []string{"/data", "-nope"})
	require.NoError(t, err)
	require.Equal(t, []string{"/data", "-nope"}, got)
}
