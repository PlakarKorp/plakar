package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/exitcodes"
	"github.com/stretchr/testify/require"
)

const plaintextEnv = "PLAKAR_INSECURE_PLAINTEXT"

// consent is the variable being present, so it has to be removed rather than
// blanked
func withoutPlaintextConsent(t *testing.T) {
	t.Helper()
	if v, ok := os.LookupEnv(plaintextEnv); ok {
		t.Setenv(plaintextEnv, v) // restored on cleanup
		os.Unsetenv(plaintextEnv)
	}
}

// a plaintext CONFIG dropped over an encrypted one would silently turn every
// later backup into cleartext, so opening such a repository has to stop
func TestEntryPointPlaintextRepositoryRefused(t *testing.T) {
	withoutPlaintextConsent(t)
	repoDir := filepath.Join(t.TempDir(), "repo")

	// creating one is not gated: -plaintext already says it out loud
	status, _, stderr := runEntryPoint(t, "at", repoDir, "create", "-plaintext")
	require.Equalf(t, 0, status, "create stderr: %s", stderr)

	status, _, stderr = runEntryPoint(t, "at", repoDir, "info")
	require.Equal(t, exitcodes.AuthFailure, status)
	require.Contains(t, stderr, "refusing to open a plaintext repository")

	t.Setenv(plaintextEnv, "1")
	status, _, stderr = runEntryPoint(t, "at", repoDir, "info")
	require.Equalf(t, 0, status, "info stderr: %s", stderr)
}

func TestEntryPointEncryptedRepositoryNeedsNoConsent(t *testing.T) {
	withoutPlaintextConsent(t)

	repoDir := filepath.Join(t.TempDir(), "repo")
	keyfile := filepath.Join(t.TempDir(), "key")
	require.NoError(t, os.WriteFile(keyfile, []byte("aZeRtY123456$#@!@\n"), 0600))

	status, _, stderr := runEntryPoint(t, "-keyfile", keyfile, "at", repoDir, "create")
	require.Equalf(t, 0, status, "create stderr: %s", stderr)

	status, _, stderr = runEntryPoint(t, "-keyfile", keyfile, "at", repoDir, "info")
	require.Equalf(t, 0, status, "info stderr: %s", stderr)
}
