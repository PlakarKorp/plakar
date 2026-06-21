package sync

import (
	"testing"

	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/stretchr/testify/require"
)

// TestRegisteredFactory looks the command up through the registry, which
// invokes the factory closure registered in init().
func TestRegisteredFactory(t *testing.T) {
	cmd, _, _, _ := subcommands.Lookup([]string{"sync"})
	require.NotNil(t, cmd)
}
