package subcommands

import (
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// fakeCmd is a minimal Subcommand for exercising the registry.
type fakeCmd struct {
	SubcommandBase
	use string
}

func (c *fakeCmd) CobraCommand() *cobra.Command { return &cobra.Command{Use: c.use} }

func (c *fakeCmd) Parse(ctx *appcontext.AppContext, args []string) error { return nil }
func (c *fakeCmd) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	return 0, nil
}

func TestSubcommandBaseFlagsAndSecret(t *testing.T) {
	cmd := &fakeCmd{}
	cmd.setFlags(BeforeRepositoryOpen)
	require.Equal(t, BeforeRepositoryOpen, cmd.GetFlags())

	cmd.RepositorySecret = []byte("secret")
	require.Equal(t, []byte("secret"), cmd.GetRepositorySecret())
}

func TestRegisterAndLookup(t *testing.T) {
	Register(func() Subcommand { return &fakeCmd{} }, NeedRepositoryKey, "ut-fake", "alpha")

	// Exact match returns a fresh command with the registered flags applied,
	// plus the consumed and remaining args.
	cmd, matched, rest := Lookup([]string{"ut-fake", "alpha", "extra1", "extra2"})
	require.NotNil(t, cmd)
	require.IsType(t, &fakeCmd{}, cmd)
	require.Equal(t, []string{"ut-fake", "alpha"}, matched)
	require.Equal(t, []string{"extra1", "extra2"}, rest)
	require.Equal(t, NeedRepositoryKey, cmd.GetFlags())
}

func TestLookupNoMatch(t *testing.T) {
	cmd, matched, rest := Lookup([]string{"ut-does-not-exist"})
	require.Nil(t, cmd)
	require.Nil(t, matched)
	require.Equal(t, []string{"ut-does-not-exist"}, rest)
}

func TestLookupTooFewArgs(t *testing.T) {
	Register(func() Subcommand { return &fakeCmd{} }, 0, "ut-twoword", "sub")

	// Fewer arguments than the registered command's arity must not match.
	cmd, _, _ := Lookup([]string{"ut-twoword"})
	require.Nil(t, cmd)
}

func TestRegisterPanicsOnZeroArgs(t *testing.T) {
	require.Panics(t, func() {
		Register(func() Subcommand { return &fakeCmd{} }, 0)
	})
}

func TestListIsSortedAndContainsRegistered(t *testing.T) {
	Register(func() Subcommand { return &fakeCmd{} }, 0, "ut-zzz")
	Register(func() Subcommand { return &fakeCmd{} }, 0, "ut-aaa")

	list := List()
	require.NotEmpty(t, list)

	// Find our two entries and confirm aaa sorts before zzz.
	var aaaIdx, zzzIdx = -1, -1
	for i, cmd := range list {
		if len(cmd) == 1 && cmd[0] == "ut-aaa" {
			aaaIdx = i
		}
		if len(cmd) == 1 && cmd[0] == "ut-zzz" {
			zzzIdx = i
		}
	}
	require.NotEqual(t, -1, aaaIdx, "ut-aaa should be listed")
	require.NotEqual(t, -1, zzzIdx, "ut-zzz should be listed")
	require.Less(t, aaaIdx, zzzIdx, "List must be sorted")
}
