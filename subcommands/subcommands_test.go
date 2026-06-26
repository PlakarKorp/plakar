package subcommands

import (
	"slices"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

type fakeCmd struct {
	SubcommandBase
	parsed bool
}

func (c *fakeCmd) Parse(ctx *appcontext.AppContext, args []string) error {
	c.parsed = true
	return nil
}
func (c *fakeCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	return 0, nil
}

func TestRegisterAndLookupSingleToken(t *testing.T) {
	Register(func() Subcommand { return &fakeCmd{} }, NeedRepositoryKey, "ut_register_single")

	cmd, matched, rest := Lookup([]string{"ut_register_single", "arg1", "arg2"})
	require.NotNil(t, cmd)
	require.Equal(t, []string{"ut_register_single"}, matched)
	require.Equal(t, []string{"arg1", "arg2"}, rest)

	// flags set by setFlags via factory closure
	require.Equal(t, NeedRepositoryKey, cmd.GetFlags())

	// factory returns the right concrete type
	fc, ok := cmd.(*fakeCmd)
	require.True(t, ok)
	require.False(t, fc.parsed)
}

func TestLookupMultiTokenPrefix(t *testing.T) {
	Register(func() Subcommand { return &fakeCmd{} }, 0, "ut_multi", "sub")

	// exact multi-token match consumes both tokens
	cmd, matched, rest := Lookup([]string{"ut_multi", "sub", "tail"})
	require.NotNil(t, cmd)
	require.Equal(t, []string{"ut_multi", "sub"}, matched)
	require.Equal(t, []string{"tail"}, rest)

	// only first token present -> no match (nargs > provided)
	cmd2, matched2, rest2 := Lookup([]string{"ut_multi"})
	require.Nil(t, cmd2)
	require.Nil(t, matched2)
	require.Equal(t, []string{"ut_multi"}, rest2)
}

func TestLookupUnknownReturnsNil(t *testing.T) {
	cmd, matched, rest := Lookup([]string{"ut_definitely_not_registered_xyz"})
	require.Nil(t, cmd)
	require.Nil(t, matched)
	require.Equal(t, []string{"ut_definitely_not_registered_xyz"}, rest)
}

func TestLookupEmptyArgs(t *testing.T) {
	cmd, _, rest := Lookup([]string{})
	require.Nil(t, cmd)
	require.Empty(t, rest)
}

func TestRegisterZeroArgsPanics(t *testing.T) {
	require.PanicsWithValue(t, "can't register commands with zero arguments", func() {
		Register(func() Subcommand { return &fakeCmd{} }, 0)
	})
}

func TestListIncludesRegistered(t *testing.T) {
	Register(func() Subcommand { return &fakeCmd{} }, 0, "ut_list_marker")
	list := List()
	require.NotEmpty(t, list)
	found := slices.ContainsFunc(list, func(args []string) bool {
		return len(args) == 1 && args[0] == "ut_list_marker"
	})
	require.True(t, found)
}

func TestSubcommandBaseSecret(t *testing.T) {
	base := &SubcommandBase{RepositorySecret: []byte("topsecret")}
	require.Equal(t, []byte("topsecret"), base.GetRepositorySecret())
	base.setFlags(BeforeRepositoryOpen)
	require.Equal(t, BeforeRepositoryOpen, base.GetFlags())
}
