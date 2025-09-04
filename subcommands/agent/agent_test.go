package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/ls"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
	os.Unsetenv("PLAKAR_AGENTLESS")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	return repo, snap, ctx
}

func initContext(t *testing.T, bufout *bytes.Buffer, buferr *bytes.Buffer) (*appcontext.AppContext, string) {
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	tmpLogDir, err := os.MkdirTemp("", "tmp_log")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpLogDir)
	})
	ctx := appcontext.NewAppContext()
	if bufout != nil && buferr != nil {
		ctx.Stdout = bufout
		ctx.Stderr = buferr
	}
	cache := caching.NewManager(tmpCacheDir)
	ctx.SetCache(cache)
	ctx.MaxConcurrency = 1
	ctx.Client = "plakar-test/1.0.0"

	var logger *logging.Logger
	if bufout == nil || buferr == nil {
		logger = logging.NewLogger(os.Stdout, os.Stderr)
	} else {
		logger = logging.NewLogger(bufout, buferr)
	}
	logger.EnableInfo()
	ctx.SetLogger(logger)

	return ctx, tmpLogDir
}

func _TestCmdAgentForegroundInit(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	ctx, logDirectory := initContext(t, bufOut, bufErr)
	logFile := filepath.Join(logDirectory, "agent.log")
	defer os.Remove(logFile)

	args := []string{"-foreground", "-log", logFile}
	subcommand := &AgentStart{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	require.Equal(t, filepath.Join(ctx.CacheDir, "agent.sock"), subcommand.socketPath)
	defer subcommand.Close()

	_, err = os.Stat(logFile)
	require.NoError(t, err)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				require.Equal(t, "unexpected call to os.Exit(0) during test", r)
			}
		}()

		status, err := subcommand.Execute(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, 0, status)
	}()

	time.Sleep(300 * time.Millisecond)

	defer func() {
		if r := recover(); r != nil {
			require.Equal(t, "unexpected call to os.Exit(0) during test", r)
		}
	}()

	client, err := agent.NewClient(filepath.Join(ctx.CacheDir, "agent.sock"), false)
	require.NoError(t, err)
	defer client.Close()

	repo, snap, ctx2 := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx2.MaxConcurrency = 1

	location, err := repo.Location()
	require.NoError(t, err)
	retval, err := client.SendCommand(ctx2, []string{"ls"}, &ls.Ls{LocateOptions: locate.NewDefaultLocateOptions(), SubcommandBase: subcommands.SubcommandBase{Flags: subcommands.AgentSupport}}, map[string]string{"location": location})
	require.NoError(t, err)
	require.Equal(t, 0, retval)

	// disabled for now because if raises: write unix @->agent.sock: write: broken pipe
	// backupDir := snap.Header.GetSource(0).Importer.Directory
	// retval, err = client.SendCommand(ctx2, []string{"cat"}, &cat.Cat{Paths: []string{filepath.Join(backupDir, "subdir/dummy.txt")}, SubcommandBase: subcommands.SubcommandBase{Flags: subcommands.AgentSupport}}, map[string]string{"location": repo.Location()})
	// require.NoError(t, err)
	// require.Equal(t, 0, retval)
}
