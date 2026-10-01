package cookies

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test creating a new manager
	manager, err := NewManager(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Verify the cookies directory was created with correct permissions
	info, err := os.Stat(filepath.Join(tmpDir, "cookies", COOKIES_VERSION))
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func hasAuthTokenFile(c *Manager) bool {
	_, err := os.Stat(filepath.Join(c.cookiesDir, ".auth-token"))
	return err == nil
}

func TestAuthTokenOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Test initial state
	hasToken := hasAuthTokenFile(manager)
	require.False(t, hasToken)

	// Test putting and getting auth token
	token := "test-auth-token"
	err = manager.PutAuthToken(token)
	require.NoError(t, err)

	hasToken = hasAuthTokenFile(manager)
	require.True(t, hasToken)

	retrievedToken, err := manager.GetAuthToken()
	require.NoError(t, err)
	require.Equal(t, token, retrievedToken)

	// Test deleting auth token
	err = manager.DeleteAuthToken()
	require.NoError(t, err)

	hasToken = hasAuthTokenFile(manager)
	require.False(t, hasToken)

	// Test getting non-existent token
	_, err = manager.GetAuthToken()
	require.Error(t, err)
}

func TestRepositoryCookieOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)
	repoID := uuid.New()
	cookieName := "test/cookie"

	// Test initial state
	hasCookie := manager.HasRepositoryCookie(repoID, cookieName)
	require.False(t, hasCookie)

	// Test putting repository cookie
	err = manager.PutRepositoryCookie(repoID, cookieName)
	require.NoError(t, err)

	// Verify cookie was created with correct name (slashes replaced with underscores)
	hasCookie = manager.HasRepositoryCookie(repoID, cookieName)
	require.True(t, hasCookie)

	// Verify the cookie file exists
	_, err = os.Stat(filepath.Join(tmpDir, "cookies", COOKIES_VERSION, repoID.String(), "test_cookie"))
	require.NoError(t, err)
}

func TestFirstRunOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Test initial state
	isFirstRun := manager.IsFirstRun()
	require.True(t, isFirstRun)

	// Test setting first run
	err = manager.SetFirstRun()
	require.NoError(t, err)

	isFirstRun = manager.IsFirstRun()
	require.False(t, isFirstRun)
}

func TestNewManagerMkdirFails(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Point the base dir at a path whose parent is a regular file, so MkdirAll
	// fails with ENOTDIR and NewManager reports it.
	blocker := filepath.Join(tmpDir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte{}, 0600))

	manager, err := NewManager(filepath.Join(blocker, "sub"))
	require.Error(t, err)
	require.Nil(t, manager)
	require.Contains(t, err.Error(), "cannot create cookies directory")
}

func TestCloseAndGetDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	require.NoError(t, manager.Close())
	require.Equal(t, filepath.Join(tmpDir, "cookies", COOKIES_VERSION), manager.GetDir())
}

func TestGetAuthTokenFromEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	t.Setenv("PLAKAR_TOKEN", "env-token")
	token, err := manager.GetAuthToken()
	require.NoError(t, err)
	require.Equal(t, "env-token", token)
}

func TestGetAuthTokenEmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// An empty token file is treated as "no token".
	require.NoError(t, manager.PutAuthToken(""))
	_, err = manager.GetAuthToken()
	require.Error(t, err)
}

func TestGetAuthTokenReadError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Make .auth-token a directory so ReadFile fails with a non-NotExist error.
	require.NoError(t, os.Mkdir(filepath.Join(manager.cookiesDir, ".auth-token"), 0700))
	_, err = manager.GetAuthToken()
	require.Error(t, err)
	require.False(t, os.IsNotExist(err))
}

func TestDeleteAuthTokenFromEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	t.Setenv("PLAKAR_TOKEN", "env-token")
	err = manager.DeleteAuthToken()
	require.ErrorIs(t, err, ErrDeleteEnvToken)
}

func TestDeleteAuthTokenNotLoggedIn(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// No token file present: deleting reports ErrNotLoggedIn.
	err = manager.DeleteAuthToken()
	require.ErrorIs(t, err, ErrNotLoggedIn)
}

func TestPutRepositoryCookieMkdirError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)
	repoID := uuid.New()

	// Create a regular file where the per-repository directory is expected, so
	// MkdirAll fails.
	require.NoError(t, os.WriteFile(filepath.Join(manager.cookiesDir, repoID.String()), []byte{}, 0600))
	err = manager.PutRepositoryCookie(repoID, "cookie")
	require.Error(t, err)
}

func TestIsFirstRunStatError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Make .first-run unstattable by placing it under a non-directory path
	// component: create a file "blocker" and point cookiesDir's child through it.
	// Simpler: replace cookiesDir with a path whose parent is a file, forcing a
	// non-NotExist stat error (ENOTDIR).
	blocker := filepath.Join(tmpDir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte{}, 0600))
	manager.cookiesDir = filepath.Join(blocker, "sub")

	require.False(t, manager.IsFirstRun())
}

func TestSecurityCheckOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager, err := NewManager(tmpDir)
	require.NoError(t, err)

	// Test initial state
	isDisabled := manager.IsDisabledSecurityCheck()
	require.False(t, isDisabled)

	// Test setting disabled security check
	err = manager.SetDisabledSecurityCheck()
	require.NoError(t, err)

	isDisabled = manager.IsDisabledSecurityCheck()
	require.True(t, isDisabled)

	// Test removing disabled security check
	err = manager.RemoveDisabledSecurityCheck()
	require.NoError(t, err)

	isDisabled = manager.IsDisabledSecurityCheck()
	require.False(t, isDisabled)
}
