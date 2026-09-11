package version

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/stretchr/testify/require"
)

func TestVersionRegisteredFactory(t *testing.T) {
	// Looking the command up through the registry invokes the factory closure
	// registered in init().
	cmd, _, _ := subcommands.Lookup([]string{"version"})
	require.NotNil(t, cmd)
	require.IsType(t, &Version{}, cmd)
}

func TestParseCmdVersion(t *testing.T) {
	ctx := &appcontext.AppContext{}
	args := []string{}

	subcommand := &Version{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
}

func TestParseCmdVersionTooManyArgs(t *testing.T) {
	ctx := &appcontext.AppContext{}
	subcommand := &Version{}
	err := subcommand.Parse(ctx, []string{"extra"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

func TestExecuteCmdVersion(t *testing.T) {
	// Create a pipe to capture stdout
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	ctx := &appcontext.AppContext{}
	repo := &repository.Repository{}

	subcommand := &Version{}
	err = subcommand.Parse(ctx, []string{})
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Close the write end of the pipe and restore stdout
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	output := buf.String()
	version := strings.Split(output, "\n")[0]
	require.Equal(t, fmt.Sprintf("plakar/%s", utils.GetVersion()), version)

}
