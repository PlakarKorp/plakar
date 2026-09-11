package subcommands

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestListComparatorPrefixBranchesCoverage4 registers two commands where one
// command's arg path is a strict prefix of the other's. This forces the two
// otherwise-uncovered tail branches of the List() sort comparator:
//   - i == len(a.args)  -> -1   (a is the shorter prefix)
//   - i == len(b.args)  -> +1   (b is the shorter prefix)
func TestListComparatorPrefixBranchesCoverage4(t *testing.T) {
	// Two-word command and its three-word extension sharing the same prefix,
	// forcing the sort comparator through its prefix tail branch.
	Register(func() Subcommand { return &fakeCmd{} }, 0, "ut-cov4", "alpha")
	Register(func() Subcommand { return &fakeCmd{} }, 0, "ut-cov4", "alpha", "beta")

	list := List()

	idxShort := slices.IndexFunc(list, func(a []string) bool {
		return slices.Equal(a, []string{"ut-cov4", "alpha"})
	})
	idxLong := slices.IndexFunc(list, func(a []string) bool {
		return slices.Equal(a, []string{"ut-cov4", "alpha", "beta"})
	})

	require.NotEqual(t, -1, idxShort, "short command must be listed")
	require.NotEqual(t, -1, idxLong, "long command must be listed")
	// The shorter prefix sorts before its extension (comparator returns -1).
	require.Less(t, idxShort, idxLong)
}

// TestLookupTooFewArgsContinueCoverage4 exercises the `nargs < subcmd.nargs`
// continue path in Lookup with a multi-word command and a single argument.
func TestLookupTooFewArgsContinueCoverage4(t *testing.T) {
	Register(func() Subcommand { return &fakeCmd{} }, 0, "ut-cov4-multi", "needstwo")

	// Only one argument provided for a two-word command -> no match.
	cmd, matched, rest := Lookup([]string{"ut-cov4-multi"})
	require.Nil(t, cmd)
	require.Nil(t, matched)
	require.Equal(t, []string{"ut-cov4-multi"}, rest)
}
