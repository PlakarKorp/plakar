package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/stretchr/testify/require"
)

// clearHomeEnv removes every variable that os.UserHomeDir / the dir helpers
// consult so that GetCacheDir/GetConfigDir/GetDataDir fall into the
// os.UserHomeDir error branch.
func clearHomeEnvCov80(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"HOME", "XDG_CACHE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME",
		"LocalAppData", "USERPROFILE", "HOMEDRIVE", "HOMEPATH",
	} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestGetCacheDirHomeErrorCov80(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("home-error branch not reachable the same way on windows")
	}
	clearHomeEnvCov80(t)
	_, err := GetCacheDir("plakar")
	require.Error(t, err)
}

func TestGetConfigDirHomeErrorCov80(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("home-error branch not reachable the same way on windows")
	}
	clearHomeEnvCov80(t)
	_, err := GetConfigDir("plakar")
	require.Error(t, err)
}

func TestGetDataDirHomeErrorCov80(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("home-error branch not reachable the same way on windows")
	}
	clearHomeEnvCov80(t)
	_, err := GetDataDir("plakar")
	require.Error(t, err)
}

// shouldCheckUpdate returns false immediately because the test binary's VERSION
// is a "-devel" build; exercise that early-return branch explicitly.
func TestShouldCheckUpdateDevelEarlyReturnCov80(t *testing.T) {
	require.Contains(t, VERSION, "devel")
	dir := t.TempDir()
	require.False(t, shouldCheckUpdate(dir))
	// No cookie should have been created since we returned before touching disk.
	_, err := os.Stat(filepath.Join(dir, "last-update-check"))
	require.True(t, os.IsNotExist(err))
}

// Set on a *time.Time field with an invalid value must surface an error
// (covers the setTime error branch via Set).
func TestPolicySetTimeInvalidCov80(t *testing.T) {
	c := &policiesConfig{Policies: map[string]*locate.LocateOptions{}}
	c.Add("p")
	err := c.Set("p", "before", "not-a-real-time")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid value")
}

// Set on a *bool field with a non-boolean value must error.
func TestPolicySetBoolInvalidCov80(t *testing.T) {
	c := &policiesConfig{Policies: map[string]*locate.LocateOptions{}}
	c.Add("p")
	err := c.Set("p", "latest", "notabool")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid value")
}

// Dump to a writer that fails should surface the encode error.
func TestPolicyDumpEncodeErrorCov80(t *testing.T) {
	c := &policiesConfig{Policies: map[string]*locate.LocateOptions{}}
	c.Add("p")
	err := c.Dump(failWriterCov80{}, "json", []string{"p"})
	require.Error(t, err)
}

type failWriterCov80 struct{}

func (failWriterCov80) Write(p []byte) (int, error) {
	return 0, errFailWriterCov80
}

var errFailWriterCov80 = &dumpWriteErr{}

type dumpWriteErr struct{}

func (*dumpWriteErr) Error() string { return "boom" }
