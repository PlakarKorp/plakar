package backup

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

type countingImporter struct {
	importer.Importer
	scans *atomic.Int64
}

func (c *countingImporter) Import(ctx context.Context, records chan<- *connectors.Record, results <-chan *connectors.Result) error {
	c.scans.Add(1)
	return c.Importer.Import(ctx, records, results)
}

func setupCountingBackup(t *testing.T, proto string) (*repository.Repository, *appcontext.AppContext, *atomic.Int64) {
	t.Helper()

	scans := &atomic.Int64{}
	err := importer.Register(proto, 0, func(ctx context.Context, opts *connectors.Options, name string, config map[string]string) (importer.Importer, error) {
		inner, err := ptesting.NewMockImporter(ctx, opts, name, config)
		if err != nil {
			return nil, err
		}
		inner.(*ptesting.MockImporter).SetFiles([]ptesting.MockFile{
			ptesting.NewMockFile("/subdir/dummy.txt", 0644, "hello dummy"),
		})
		return &countingImporter{Importer: inner, scans: scans}, nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, importer.Unregister(proto)) })

	repo, _, ctx := generateFixtures(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	renderer := stdio.New(ctx)
	require.NoError(t, renderer.Run())
	t.Cleanup(func() { _ = renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	return repo, ctx, scans
}

func runCountingBackup(t *testing.T, repo *repository.Repository, ctx *appcontext.AppContext, args ...string) {
	t.Helper()

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, args))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupSkipsFilesystemSummaryWhenNothingConsumesIt(t *testing.T) {
	repo, ctx, scans := setupCountingBackup(t, "countsummaryoff")

	runCountingBackup(t, repo, ctx, "countsummaryoff://tree")

	require.Equal(t, int64(1), scans.Load(),
		"backup must walk the source once when no renderer consumes the filesystem summary")
}

func TestBackupComputesFilesystemSummaryForProgressRenderer(t *testing.T) {
	repo, ctx, scans := setupCountingBackup(t, "countsummaryon")
	ctx.ProgressSummary = true

	runCountingBackup(t, repo, ctx, "countsummaryon://tree")

	require.Equal(t, int64(2), scans.Load(),
		"backup must still gather the filesystem summary for a renderer that displays it")
}

func TestBackupSkipsFilesystemSummaryWithNoProgress(t *testing.T) {
	repo, ctx, scans := setupCountingBackup(t, "countsummarynoprogress")
	ctx.ProgressSummary = true

	runCountingBackup(t, repo, ctx, "-no-progress", "countsummarynoprogress://tree")

	require.Equal(t, int64(1), scans.Load(),
		"-no-progress must keep the source from being walked twice")
}
