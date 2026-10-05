package expose

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/visualizer"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

type testVisualizer struct {
	exposed bool
}

func (v *testVisualizer) Expose(context.Context, *vfs.Filesystem) error {
	v.exposed = true
	return nil
}

func registerTestVisualizer(t *testing.T, protocol string, instance *testVisualizer) {
	t.Helper()

	err := visualizer.Register(protocol, 0, func(context.Context, *connectors.Options, string, map[string]string) (visualizer.Visualizer, error) {
		return instance, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, visualizer.Unregister(protocol))
	})
}

func TestExposeRegisteredFactory(t *testing.T) {
	cmd, _, _ := subcommands.Lookup([]string{"expose"})
	require.NotNil(t, cmd)
	require.IsType(t, &Expose{}, cmd)
}

func TestExposeParse(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Expose{}
	require.NoError(t, cmd.Parse(ctx, []string{"visualizer-test://abc123"}))
	require.Equal(t, "visualizer-test://abc123", cmd.Location)
	require.Equal(t, "abc123", cmd.SnapshotPath)
}

func TestExposeParseRejectsInvalidLocation(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	for _, args := range [][]string{
		nil,
		{"abc123"},
		{"visualizer-test://"},
		{"one://abc", "two://def"},
	} {
		cmd := &Expose{}
		require.Error(t, cmd.Parse(ctx, args))
	}
}

func TestExposeExecuteRejectsUnsupportedVisualizer(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	cmd := &Expose{}
	require.NoError(t, cmd.Parse(ctx, []string{"not-registered://abc123"}))
	status, err := cmd.Execute(ctx, repo)
	require.EqualError(t, err, "unsupported visualizer protocol")
	require.Equal(t, 1, status)
}

func TestExposeExecute(t *testing.T) {
	const protocol = "expose-visualizer-test"

	instance := &testVisualizer{}
	registerTestVisualizer(t, protocol, instance)

	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "x"),
	})
	defer snap.Close()

	cmd := &Expose{}
	location := fmt.Sprintf("%s://%x", protocol, snap.Header.Identifier)
	require.NoError(t, cmd.Parse(ctx, []string{location}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.True(t, instance.exposed)
}
