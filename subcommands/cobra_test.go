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

func TestHasHelpOpt(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
		want bool
	}{
		{name: "single dash help", in: []string{"-help"}, want: true},
		{name: "double dash help", in: []string{"--help"}, want: true},
		{name: "short help", in: []string{"-h"}, want: true},
		{name: "help after a bool flag", in: []string{"-quiet", "-h"}, want: true},
		{name: "help after a flag and its value", in: []string{"-name", "snap", "-help"}, want: true},
		{name: "help after an equals value", in: []string{"-name=snap", "-h"}, want: true},
		{name: "help after a one letter long option", in: []string{"-o", "out.txt", "-h"}, want: true},
		{
			// People append --help to the command line they just ran.
			name: "help after the first operand",
			in:   []string{"-name", "-h", "/data", "--help"},
			want: true,
		},
		{
			// Past the first operand -name is data, not an option, so it
			// does not consume the -h after it.
			name: "help after an operand that looks like a flag",
			in:   []string{"/data", "-name", "-h"},
			want: true,
		},
		{
			// The unknown option cannot be told to take a value or not;
			// parsing will reject it, help is the better answer.
			name: "help after an unknown option",
			in:   []string{"-nope", "--help"},
			want: true,
		},
		{name: "help after stdin", in: []string{"-", "-h"}, want: true},
		{name: "help as the value of a flag", in: []string{"-name", "-h", "/data"}, want: false},
		{name: "help as the value of a one letter long option", in: []string{"-o", "-help"}, want: false},
		{name: "help word after dashdash", in: []string{"--", "--help"}, want: false},
		{name: "help word after an operand and dashdash", in: []string{"/data", "--", "-h"}, want: false},
		{name: "help word inside an operand", in: []string{"./--help"}, want: false},
		{name: "help with a value is not a help word", in: []string{"-help=true"}, want: false},
		{name: "no help", in: []string{"-quiet", "-name", "snap", "/data"}, want: false},
		{name: "no arguments", in: []string{}, want: false},
		{name: "nil arguments", in: nil, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, HasHelpOpt(testFlags(), tc.in))
		})
	}
}
