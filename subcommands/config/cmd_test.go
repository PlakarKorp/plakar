package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/stretchr/testify/require"
)

// TestEntityCommands covers the four entity commands, which are the same
// Parse/Execute wrapper around dispatch: registration, the no-action parse
// error, and the status a successful and a failing dispatch return.
func TestEntityCommands(t *testing.T) {
	entities := []struct {
		kind    string
		newCmd  func() subcommands.Subcommand
		addArgs []string
		added   func(t *testing.T, ctx *appcontext.AppContext)
	}{
		{
			"store",
			func() subcommands.Subcommand { return &ConfigStoreCmd{} },
			[]string{"add", "n", "fs:/tmp/n"},
			func(t *testing.T, ctx *appcontext.AppContext) { require.True(t, ctx.Config.HasRepository("n")) },
		},
		{
			"source",
			func() subcommands.Subcommand { return &ConfigSourceCmd{} },
			[]string{"add", "n", "fs:/tmp/n"},
			func(t *testing.T, ctx *appcontext.AppContext) { require.True(t, ctx.Config.HasSource("n")) },
		},
		{
			"destination",
			func() subcommands.Subcommand { return &ConfigDestinationCmd{} },
			[]string{"add", "n", "fs:/tmp/n"},
			func(t *testing.T, ctx *appcontext.AppContext) { require.True(t, ctx.Config.HasDestination("n")) },
		},
		{
			"policy",
			func() subcommands.Subcommand { return &ConfigPolicyCmd{} },
			[]string{"add", "nightly"},
			func(t *testing.T, ctx *appcontext.AppContext) {
				_, err := os.Stat(filepath.Join(ctx.ConfigDir, "policies.yml"))
				require.NoError(t, err)
			},
		},
	}

	for _, e := range entities {
		t.Run(e.kind+"/registered", func(t *testing.T) {
			// look the command up through the registry to invoke the factory
			// closure registered in init().
			cmd, _, _ := subcommands.Lookup([]string{e.kind})
			require.NotNil(t, cmd, "command %q not registered", e.kind)
			require.IsType(t, e.newCmd(), cmd)
		})

		t.Run(e.kind+"/parse-no-action", func(t *testing.T) {
			ctx, _, _ := newCtx(t)
			require.Error(t, e.newCmd().Parse(ctx, []string{}))
		})

		t.Run(e.kind+"/add", func(t *testing.T) {
			ctx, _, _ := newCtx(t)

			cmd := e.newCmd()
			require.NoError(t, cmd.Parse(ctx, e.addArgs))
			status, err := cmd.Execute(ctx, &repository.Repository{})
			require.NoError(t, err)
			require.Equal(t, 0, status)
			e.added(t, ctx)
		})

		t.Run(e.kind+"/rm-unknown", func(t *testing.T) {
			ctx, _, _ := newCtx(t)

			// a failing dispatch propagates status 1
			cmd := e.newCmd()
			require.NoError(t, cmd.Parse(ctx, []string{"rm", "ghost"}))
			status, err := cmd.Execute(ctx, &repository.Repository{})
			require.Error(t, err)
			require.Equal(t, 1, status)
		})
	}
}
