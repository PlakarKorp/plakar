package appcontext

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/cookies"
)

func TestNewAppContext(t *testing.T) {
	ctx := NewAppContext()
	if ctx == nil {
		t.Fatal("NewAppContext returned nil")
	}
	if ctx.KContext == nil {
		t.Fatal("NewAppContext did not initialize KContext")
	}
	if ctx.GetInner() != ctx.KContext {
		t.Fatal("GetInner did not return the embedded KContext")
	}
}

func TestErrorCauseReturnsCancelCause(t *testing.T) {
	ctx := NewAppContext()
	cause := errors.New("underlying failure")
	ctx.Cancel(cause)

	if got := ctx.ErrorCause(context.Canceled); !errors.Is(got, cause) {
		t.Fatalf("ErrorCause() = %v, want %v", got, cause)
	}
}

func TestErrorCauseReturnsOriginalError(t *testing.T) {
	ctx := NewAppContext()
	err := errors.New("regular failure")

	if got := ctx.ErrorCause(err); !errors.Is(got, err) {
		t.Fatalf("ErrorCause() = %v, want %v", got, err)
	}
}

func TestSecretRoundTrip(t *testing.T) {
	ctx := NewAppContext()
	if got := ctx.GetSecret(); got != nil {
		t.Fatalf("expected nil secret on new context, got %v", got)
	}
	secret := []byte("hunter2")
	ctx.SetSecret(secret)
	if got := ctx.GetSecret(); !bytes.Equal(got, secret) {
		t.Fatalf("GetSecret = %q, want %q", got, secret)
	}
}

func TestCookiesAndPkgManagerAccessors(t *testing.T) {
	ctx := NewAppContext()
	if ctx.GetCookies() != nil {
		t.Fatal("expected nil cookies on new context")
	}
	if ctx.GetPkgManager() != nil {
		t.Fatal("expected nil pkg manager on new context")
	}

	mgr := cookies.NewManager(t.TempDir())
	ctx.SetCookies(mgr)
	if ctx.GetCookies() != mgr {
		t.Fatal("SetCookies/GetCookies mismatch")
	}

	// SetPkgManager accepts a *pkg.Manager but nil is a valid value too;
	// confirm we can store and retrieve it without panicking.
	ctx.SetPkgManager(nil)
	if ctx.GetPkgManager() != nil {
		t.Fatal("expected nil after SetPkgManager(nil)")
	}
}

func TestNewAppContextFromCopiesFields(t *testing.T) {
	parent := NewAppContext()
	parent.ConfigDir = "/tmp/cfg"
	mgr := cookies.NewManager(t.TempDir())
	parent.SetCookies(mgr)
	parent.SetSecret([]byte("s"))

	child := NewAppContextFrom(parent)
	if child == parent {
		t.Fatal("NewAppContextFrom returned the same pointer")
	}
	if child.KContext == parent.KContext {
		t.Fatal("child should have its own KContext")
	}
	if child.ConfigDir != "/tmp/cfg" {
		t.Fatalf("ConfigDir = %q, want %q", child.ConfigDir, "/tmp/cfg")
	}
	if child.GetCookies() != mgr {
		t.Fatal("child did not inherit cookies")
	}
	// secret is intentionally NOT copied — document that contract.
	if child.GetSecret() != nil {
		t.Fatalf("child.GetSecret() = %v, want nil (secret should not be inherited)", child.GetSecret())
	}
}

func TestImporterAndExporterOpts(t *testing.T) {
	ctx := NewAppContext()
	ctx.Hostname = "host"
	ctx.OperatingSystem = "linux"
	ctx.Architecture = "amd64"
	ctx.CWD = "/work"
	ctx.MaxConcurrency = 4
	stdin := bytes.NewReader(nil)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ctx.Stdin = stdin
	ctx.Stdout = stdout
	ctx.Stderr = stderr

	for name, opts := range map[string]struct {
		hostname, os, arch, cwd string
		concurrency             int
	}{
		"importer": {ctx.ImporterOpts().Hostname, ctx.ImporterOpts().OperatingSystem, ctx.ImporterOpts().Architecture, ctx.ImporterOpts().CWD, ctx.ImporterOpts().MaxConcurrency},
		"exporter": {ctx.ExporterOpts().Hostname, ctx.ExporterOpts().OperatingSystem, ctx.ExporterOpts().Architecture, ctx.ExporterOpts().CWD, ctx.ExporterOpts().MaxConcurrency},
	} {
		if opts.hostname != "host" || opts.os != "linux" || opts.arch != "amd64" || opts.cwd != "/work" || opts.concurrency != 4 {
			t.Fatalf("%s: opts mismatch: %+v", name, opts)
		}
	}

	if ctx.ImporterOpts().Stdin != stdin || ctx.ImporterOpts().Stdout != stdout || ctx.ImporterOpts().Stderr != stderr {
		t.Fatal("ImporterOpts did not propagate std streams")
	}
	if ctx.ExporterOpts().Stdin != stdin || ctx.ExporterOpts().Stdout != stdout || ctx.ExporterOpts().Stderr != stderr {
		t.Fatal("ExporterOpts did not propagate std streams")
	}
}

func TestReloadConfig(t *testing.T) {
	ctx := NewAppContext()
	ctx.ConfigDir = t.TempDir()
	if err := ctx.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig on empty dir: %v", err)
	}
	if ctx.Config == nil {
		t.Fatal("ReloadConfig left Config nil")
	}
}

func TestReloadConfig_PropagatesLoadError(t *testing.T) {
	ctx := NewAppContext()
	dir := t.TempDir()
	ctx.ConfigDir = dir

	// A sources.yml that exists but is unparseable causes LoadConfig to return
	// a non-IsNotExist error, exercising the error-propagation branch.
	if err := os.WriteFile(filepath.Join(dir, "sources.yml"),
		[]byte("not: [valid: yaml"), 0o600); err != nil {
		t.Fatalf("write malformed sources.yml: %v", err)
	}

	if err := ctx.ReloadConfig(); err == nil {
		t.Fatal("expected ReloadConfig to propagate parse error, got nil")
	}
}
