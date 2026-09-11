package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	_ "github.com/PlakarKorp/integrations/fs/importer"
	_ "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/stretchr/testify/require"
)

func cov2Ctx(t *testing.T) (*appcontext.AppContext, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	tmpDir := t.TempDir()
	cfg, err := config.LoadOldConfigIfExists(filepath.Join(tmpDir, "config.yaml"))
	require.NoError(t, err)
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	ctx := appcontext.NewAppContext()
	ctx.Config = cfg
	ctx.ConfigDir = tmpDir
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	return ctx, bufOut, bufErr
}

// ---------- Execute error path returns status 1 ----------

func TestCov2StoreExecuteErrorStatus2(t *testing.T) {
	ctx, _, _ := cov2Ctx(t)
	cmd := &ConfigStoreCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"rm", "does-not-exist"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2SourceExecuteErrorStatus2(t *testing.T) {
	ctx, _, _ := cov2Ctx(t)
	cmd := &ConfigSourceCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"rm", "nope"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2DestinationExecuteErrorStatus2(t *testing.T) {
	ctx, _, _ := cov2Ctx(t)
	cmd := &ConfigDestinationCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"rm", "nope"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2PolicyExecuteErrorStatus2(t *testing.T) {
	ctx, _, _ := cov2Ctx(t)
	cmd := &ConfigPolicyCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"rm", "nope"}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2StoreExecuteSuccessStatus2(t *testing.T) {
	ctx, _, _ := cov2Ctx(t)
	cmd := &ConfigStoreCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"add", "r", "fs://" + t.TempDir()}))
	status, err := cmd.Execute(ctx, &repository.Repository{})
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// ---------- import: overwrite, rename, skip-missing, empty-name ----------

func TestCov2ImportOverwriteAllSections2(t *testing.T) {
	ctx, _, bufErr := cov2Ctx(t)
	// seed an existing source
	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"alpha", "fs:///old"}))

	cfgFile := filepath.Join(t.TempDir(), "import.yml")
	body := "alpha:\n  location: fs:///new\nbeta:\n  location: fs:///b\n"
	require.NoError(t, os.WriteFile(cfgFile, []byte(body), 0o600))

	// without -overwrite alpha is skipped, beta added
	require.NoError(t, dispatchSubcommand(ctx, "source", "import", []string{"-config", cfgFile}))
	require.Contains(t, bufErr.String(), "already exists, skipping")
	require.Equal(t, "fs:///old", ctx.Config.Sources["alpha"]["location"])
	require.Equal(t, "fs:///b", ctx.Config.Sources["beta"]["location"])

	// with -overwrite alpha is replaced
	require.NoError(t, dispatchSubcommand(ctx, "source", "import", []string{"-overwrite", "-config", cfgFile}))
	require.Equal(t, "fs:///new", ctx.Config.Sources["alpha"]["location"])
}

func TestCov2ImportSelectedRenameAndSkips2(t *testing.T) {
	ctx, _, bufErr := cov2Ctx(t)
	cfgFile := filepath.Join(t.TempDir(), "import.yml")
	body := "alpha:\n  location: fs:///a\nbeta:\n  location: fs:///b\n"
	require.NoError(t, os.WriteFile(cfgFile, []byte(body), 0o600))

	// rename alpha->renamed, request a missing section, and an empty target
	args := []string{"-config", cfgFile, "alpha:renamed", "ghost", ":bad"}
	require.NoError(t, dispatchSubcommand(ctx, "source", "import", args))

	require.Equal(t, "fs:///a", ctx.Config.Sources["renamed"]["location"])
	errStr := bufErr.String()
	require.Contains(t, errStr, "does not exist in config")
	require.Contains(t, errStr, "empty section name")
}

func TestCov2ImportSelectedAlreadyExistsSkip2(t *testing.T) {
	ctx, _, bufErr := cov2Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"alpha", "fs:///old"}))
	cfgFile := filepath.Join(t.TempDir(), "import.yml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("alpha:\n  location: fs:///new\n"), 0o600))

	// selected import, target alpha exists, no overwrite -> skip
	require.NoError(t, dispatchSubcommand(ctx, "source", "import", []string{"-config", cfgFile, "alpha"}))
	require.Contains(t, bufErr.String(), "already exists, skipping")
	require.Equal(t, "fs:///old", ctx.Config.Sources["alpha"]["location"])
}

func TestCov2ImportEmptyConfigError2(t *testing.T) {
	ctx, _, _ := cov2Ctx(t)
	cfgFile := filepath.Join(t.TempDir(), "empty.yml")
	// valid yaml with a location but a section that becomes empty? Use a doc with
	// only a scalar top-level so GetConf returns no sections -> "no valid".
	require.NoError(t, os.WriteFile(cfgFile, []byte("location: fs:///x\n"), 0o600))
	err := dispatchSubcommand(ctx, "source", "import", []string{"-config", cfgFile})
	require.Error(t, err)
}

// ---------- show: aggregate missing-name error & default masking ----------

func TestCov2ShowMissingNameAggregateError2(t *testing.T) {
	ctx, _, bufErr := cov2Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:///x"}))
	err := dispatchSubcommand(ctx, "store", "show", []string{"r", "ghost"})
	require.Error(t, err)
	require.Contains(t, bufErr.String(), "does not exist")
}

func TestCov2ShowMasksSecretsByDefault2(t *testing.T) {
	ctx, bufOut, _ := cov2Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add",
		[]string{"r", "fs:///x", "secret_access_key=topsecret", "x_token=abc"}))
	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"r"}))
	out := bufOut.String()
	require.Contains(t, out, "********")
	require.NotContains(t, out, "topsecret")
	require.NotContains(t, out, "abc")
}

func TestCov2ShowJSONFormat2(t *testing.T) {
	ctx, bufOut, _ := cov2Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:///x"}))
	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"-json", "r"}))
	require.Contains(t, bufOut.String(), "\"location\"")
}

// ---------- unset cannot remove location ----------

func TestCov2UnsetLocationRejected2(t *testing.T) {
	ctx, _, _ := cov2Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:///x"}))
	err := dispatchSubcommand(ctx, "store", "unset", []string{"r", "location"})
	require.Error(t, err)
}

// ---------- policy dispatch default + show yaml/json ----------

func TestCov2PolicyDefaultSubcommand2(t *testing.T) {
	ctx, _, _ := cov2Ctx(t)
	err := dispatchPolicy(ctx, "policy", "bogus", nil)
	require.Error(t, err)
}

func TestCov2PolicyShowJSON2(t *testing.T) {
	ctx, bufOut, _ := cov2Ctx(t)
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"daily", "days=7"}))
	bufOut.Reset()
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"-json", "daily"}))
	require.Contains(t, bufOut.String(), "daily")
}
