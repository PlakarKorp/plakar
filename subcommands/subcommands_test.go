package subcommands

import (
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

type testCommand struct{ SubcommandBase }

func (t *testCommand) Parse(_ *appcontext.AppContext, _ []string) error {
	return nil
}

func (t *testCommand) Execute(_ *appcontext.AppContext, _ *repository.Repository) (int, error) {
	return 0, nil
}

func TestRegisterValid(t *testing.T) {
	orig := subcommands
	subcommands = nil
	t.Cleanup(func() { subcommands = orig })

	err := Register(func() Subcommand { return &testCommand{} }, 0, "foo")
	require.NoError(t, err)

	cmd, args, remaining := Lookup([]string{"foo"})
	require.NotNil(t, cmd)
	require.Equal(t, []string{"foo"}, args)
	require.Empty(t, remaining)
}

func TestRegisterValidationFailures(t *testing.T) {
	orig := subcommands
	subcommands = nil
	t.Cleanup(func() { subcommands = orig })

	factory := func() Subcommand { return &testCommand{} }

	t.Run("nil factory", func(t *testing.T) {
		err := Register(nil, 0, "foo")
		require.Error(t, err)
		require.ErrorIs(t, err, errNilFactory)
	})

	t.Run("no arguments", func(t *testing.T) {
		err := Register(factory, 0)
		require.Error(t, err)
		require.ErrorIs(t, err, errNoArguments)
	})

	t.Run("empty argument", func(t *testing.T) {
		err := Register(factory, 0, "")
		require.Error(t, err)
	})

	t.Run("spaces in argument", func(t *testing.T) {
		err := Register(factory, 0, "has space")
		require.Error(t, err)
	})

	require.NoError(t, Register(factory, 0, "foo"))

	t.Run("duplicate registration", func(t *testing.T) {
		err := Register(factory, 0, "foo")
		require.Error(t, err)
		require.ErrorIs(t, err, errDuplicateCommand)
	})
}

func TestMustRegisterPanicsOnError(t *testing.T) {
	orig := subcommands
	subcommands = nil
	t.Cleanup(func() { subcommands = orig })

	require.Panics(t, func() {
		MustRegister(nil, 0, "foo")
	})
}
