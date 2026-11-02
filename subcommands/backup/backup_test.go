package backup

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/importer"
	bfs "github.com/PlakarKorp/integrations/fs/storage"
	_ "github.com/PlakarKorp/integrations/tar/importer"
	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/caching/pebble"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateFixtures(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, string, *appcontext.AppContext) {
	// See comment in backup.go
	stateRefresher = func(*appcontext.AppContext, *repository.Repository) func(objects.MAC, bool) error {
		return nil
	}

	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_repo")
	require.NoError(t, err)
	tmpRepoDir := fmt.Sprintf("%s/repo", tmpRepoDirRoot)
	tmpCacheDir, err := os.MkdirTemp("", "tmp_cache")
	require.NoError(t, err)
	tmpBackupDir, err := os.MkdirTemp("", "tmp_to_backup")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpCacheDir)
		os.RemoveAll(tmpBackupDir)
		os.RemoveAll(tmpRepoDirRoot)
	})
	// create temporary files to backup
	err = os.MkdirAll(tmpBackupDir+"/subdir", 0755)
	require.NoError(t, err)
	err = os.MkdirAll(tmpBackupDir+"/another_subdir", 0755)
	require.NoError(t, err)
	err = os.WriteFile(tmpBackupDir+"/subdir/dummy.txt", []byte("hello dummy"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(tmpBackupDir+"/subdir/foo.txt", []byte("hello foo"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(tmpBackupDir+"/subdir/to_exclude", []byte("*/subdir/to_exclude\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(tmpBackupDir+"/another_subdir/bar", []byte("hello bar"), 0644)
	require.NoError(t, err)

	ctx := appcontext.NewAppContext()

	// create a storage
	r, err := bfs.NewStore(ctx, "fs", map[string]string{"location": tmpRepoDir})
	require.NotNil(t, r)
	require.NoError(t, err)
	config := storage.NewConfiguration()
	serialized, err := config.ToBytes()
	require.NoError(t, err)

	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)
	wrappedConfigRd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serialized))
	require.NoError(t, err)

	wrappedConfig, err := io.ReadAll(wrappedConfigRd)
	require.NoError(t, err)

	err = r.Create(ctx, wrappedConfig)
	require.NoError(t, err)

	// open the storage to load the configuration
	r, serializedConfig, err := storage.Open(ctx.GetInner(), map[string]string{"location": "fs://" + tmpRepoDir})
	require.NoError(t, err)

	// create a repository
	cache := caching.NewManager(pebble.Constructor(tmpCacheDir))
	ctx.SetCache(cache)
	ctx.CacheDir = tmpCacheDir
	ctx.Client = "plakar-test/1.0.0"

	// Create a new logger
	logger := logging.NewLogger(bufOut, bufErr)
	// logger := logging.NewLogger(os.Stdout, os.Stderr)
	logger.EnableInfo()
	// logger.EnableTrace("all")
	ctx.SetLogger(logger)
	repo, err := repository.New(ctx.GetInner(), nil, r, serializedConfig)
	require.NoError(t, err, "creating repository")

	return repo, tmpBackupDir, ctx
}

func TestExecuteCmdCreateDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()

	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	args := []string{tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should be something like:
	// info: 9a383818: OK ✓ /tmp/tmp_to_backup2103009254/subdir/foo.txt
	// info: 9a383818: OK ✓ /tmp/tmp_to_backup2103009254/another_subdir/bar
	// info: 9a383818: OK ✓ /tmp/tmp_to_backup2103009254/subdir/dummy.txt
	// info: 9a383818: OK ✓ /tmp/tmp_to_backup2103009254/subdir/to_exclude
	// info: 9a383818: OK ✓ /tmp/tmp_to_backup2103009254/subdir
	// info: 9a383818: OK ✓ /tmp/tmp_to_backup2103009254/another_subdir
	// info: 9a383818: OK ✓ /tmp/tmp_to_backup2103009254
	// info: 9a383818: OK ✓ /tmp
	// info: 9a383818: OK ✓ /
	// info: created unsigned snapshot 9a383818 with root PoRwWDCajeHqDG0vkZu13jOAWo3U/Wr9e/Hecg4IJoU of size 29 B in 10.961071ms

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	// last line should have the summary
	lastline := lines[len(lines)-1]
	require.Contains(t, lastline, "backup completed")
}

func TestExecuteCmdCreateWithHooks(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	args := []string{tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	// Set hooks
	subcommand.PreHook = "echo 'pre-hook executed'"
	subcommand.PostHook = "echo 'post-hook executed'"

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, "executing hook: echo 'pre-hook executed'")
	require.Contains(t, output, "pre-hook executed")
	require.Contains(t, output, "executing hook: echo 'post-hook executed'")
	require.Contains(t, output, "post-hook executed")
	require.Contains(t, output, "backup completed")
}

func TestBackupWithPlakarTagsEnv(t *testing.T) {
	original := os.Getenv("PLAKAR_TAGS")
	defer os.Setenv("PLAKAR_TAGS", original)
	os.Setenv("PLAKAR_TAGS", "daily,important")

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	args := []string{tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)

	require.Equal(t, []string{"daily", "important"}, subcommand.Tags)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupTagFlagOverridesEnv(t *testing.T) {
	original := os.Getenv("PLAKAR_TAGS")
	defer os.Setenv("PLAKAR_TAGS", original)
	os.Setenv("PLAKAR_TAGS", "env-tag1,env-tag2")

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	args := []string{"-tag", "cli-tag", tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)

	// CLI flag should win over env var
	require.Equal(t, []string{"cli-tag"}, subcommand.Tags)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupEmptyPlakarTagsEnv(t *testing.T) {
	original := os.Getenv("PLAKAR_TAGS")
	defer os.Setenv("PLAKAR_TAGS", original)
	os.Setenv("PLAKAR_TAGS", "")

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	args := []string{tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)

	// No tags should be set
	require.Equal(t, []string{}, subcommand.Tags)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupPlakarTagsWhitespace(t *testing.T) {
	original := os.Getenv("PLAKAR_TAGS")
	defer os.Setenv("PLAKAR_TAGS", original)
	os.Setenv("PLAKAR_TAGS", "ci, nightly , prod")

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	args := []string{tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)

	require.Equal(t, []string{"ci", "nightly", "prod"}, subcommand.Tags)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupPlakarTagsDoubleComma(t *testing.T) {
	original := os.Getenv("PLAKAR_TAGS")
	defer os.Setenv("PLAKAR_TAGS", original)
	os.Setenv("PLAKAR_TAGS", "ci,,nightly")

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	args := []string{tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)

	require.Equal(t, []string{"ci", "nightly"}, subcommand.Tags)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestBackupPlakarTagsTrailingComma(t *testing.T) {
	original := os.Getenv("PLAKAR_TAGS")
	defer os.Setenv("PLAKAR_TAGS", original)
	os.Setenv("PLAKAR_TAGS", "ci,nightly,")

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	defer renderer.Wait()
	defer ctx.Close()

	ctx.MaxConcurrency = 1
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	args := []string{tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)

	require.Equal(t, []string{"ci", "nightly"}, subcommand.Tags)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

func TestExecuteCmdCreateDefaultWithIgnores(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

	ctx.MaxConcurrency = 1
	args := []string{"-ignore", "**/subdir", tmpBackupDir}

	subcommand := &Backup{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.NotContains(t, output, "/subdir")
}

func TestBackupSourceIgnoreRules(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options func(t *testing.T) map[string]string
		args    []string
		absent  []string
		present string
	}{
		{
			name: "ignore pattern",
			options: func(*testing.T) map[string]string {
				return map[string]string{"ignore": "**/subdir"}
			},
			absent:  []string{"/subdir/"},
			present: "/another_subdir/",
		},
		{
			name: "ignore list",
			options: func(*testing.T) map[string]string {
				return map[string]string{"ignore": "**/nothing,**/subdir"}
			},
			absent:  []string{"/subdir/"},
			present: "/another_subdir/",
		},
		{
			name: "ignore file",
			options: func(t *testing.T) map[string]string {
				path := filepath.Join(t.TempDir(), "ignores")
				require.NoError(t, os.WriteFile(path, []byte("# comment\n\n**/subdir\n"), 0o600))
				return map[string]string{"ignore-file": path}
			},
			absent:  []string{"/subdir/"},
			present: "/another_subdir/",
		},
		{
			name: "command line rules apply on top",
			options: func(*testing.T) map[string]string {
				return map[string]string{"ignore": "**/another_subdir"}
			},
			args:    []string{"-ignore", "**/subdir"},
			absent:  []string{"/subdir/", "/another_subdir/"},
			present: "backup completed",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bufOut := bytes.NewBuffer(nil)
			bufErr := bytes.NewBuffer(nil)
			repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)

			renderer := stdio.New(ctx)
			renderer.Run()
			t.Cleanup(func() { renderer.Wait() })
			t.Cleanup(ctx.Close)
			ctx.MaxConcurrency = 1

			ctx.Config = config.NewConfig()
			source := map[string]string{"location": "fs:" + tmpBackupDir}
			maps.Copy(source, tc.options(t))
			ctx.Config.Sources["configured"] = source

			cmd := &Backup{}
			require.NoError(t, cmd.Parse(ctx, append(tc.args, "@configured")))

			status, err := cmd.Execute(ctx, repo)
			require.NoError(t, err)
			require.Equal(t, 0, status)

			out := bufOut.String()
			for _, absent := range tc.absent {
				require.NotContains(t, out, absent, "the configured ignore rules were not applied")
			}
			require.Contains(t, out, tc.present)
		})
	}
}

func TestBackupSourceIgnoreFileMissing(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	ctx.Config = config.NewConfig()
	ctx.Config.Sources["configured"] = map[string]string{
		"location":    "fs:" + tmpBackupDir,
		"ignore-file": filepath.Join(t.TempDir(), "does-not-exist"),
	}

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"@configured"}))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "source @configured")
}

func TestBackupSourceIgnoreRulesReachNonFsImporter(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, _, ctx := generateFixtures(t, bufOut, bufErr)

	renderer := stdio.New(ctx)
	renderer.Run()
	t.Cleanup(func() { renderer.Wait() })
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	archive := filepath.Join(t.TempDir(), "src.tar")
	fp, err := os.Create(archive)
	require.NoError(t, err)
	tw := tar.NewWriter(fp)
	for _, name := range []string{"keepme.txt", "skipme.txt"} {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o644, Size: int64(len(name)), Typeflag: tar.TypeReg,
		}))
		_, err = tw.Write([]byte(name))
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, fp.Close())

	ctx.Config = config.NewConfig()
	ctx.Config.Sources["archive"] = map[string]string{
		"location": "tar:" + archive,
		"ignore":   "**/skipme.txt",
	}

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"@archive"}))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "keepme.txt")
	require.NotContains(t, out, "skipme.txt")
}

func TestBackupSourceIgnoreRulesConflictInSameOrigin(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, tmpBackupDir, ctx := generateFixtures(t, bufOut, bufErr)
	t.Cleanup(ctx.Close)
	ctx.MaxConcurrency = 1

	ctx.Config = config.NewConfig()
	ctx.Config.Sources["first"] = map[string]string{
		"location": "fs:" + filepath.Join(tmpBackupDir, "subdir"),
		"ignore":   "**/dummy.txt",
	}
	ctx.Config.Sources["second"] = map[string]string{
		"location": "fs:" + filepath.Join(tmpBackupDir, "another_subdir"),
		"ignore":   "**/bar",
	}

	cmd := &Backup{}
	require.NoError(t, cmd.Parse(ctx, []string{"@first", "@second"}))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "ignore rules differ from another source of the same origin")
}
