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
	manager := NewManager(tmpDir)
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

	manager := NewManager(tmpDir)

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

	manager := NewManager(tmpDir)
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

	manager := NewManager(tmpDir)

	// Test initial state
	isFirstRun := manager.IsFirstRun()
	require.True(t, isFirstRun)

	// Test setting first run
	err = manager.SetFirstRun()
	require.NoError(t, err)

	isFirstRun = manager.IsFirstRun()
	require.False(t, isFirstRun)
}

func TestSecurityCheckOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cookies_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

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
