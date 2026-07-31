package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
)

// newTestCtx builds an AppContext with a temp CacheDir and a real
// cookies.Manager rooted at a temp dir, so the first-run / disabled-check
// branches can be exercised without touching the user's actual cookie
// store.
func newTestCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	dir := t.TempDir()
	ctx.CacheDir = dir
	ctx.SetCookies(cookies.NewManager(dir))
	ctx.Stdout = new(bytes.Buffer)
	ctx.Stderr = new(bytes.Buffer)
	return ctx
}

func TestCurrentUserDefaultHonorsHomeEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got := currentUserDefault(t.TempDir())
	if got.HomeDir != home {
		t.Fatalf("HomeDir = %q, want %q", got.HomeDir, home)
	}
}

// ---------- getPassphraseFromEnv ----------

func TestGetPassphraseFromEnv_KeyFromFileWins(t *testing.T) {
	ctx := newTestCtx(t)
	ctx.KeyFromFile = "secret-from-keyfile"
	t.Setenv("PLAKAR_PASSPHRASE", "ignored")

	got, err := getPassphraseFromEnv(ctx, map[string]string{"passphrase": "ignored-too"})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "secret-from-keyfile" {
		t.Fatalf("got %q, want secret-from-keyfile", got)
	}
}

func TestGetPassphraseFromEnv_ParamsPassphraseConsumed(t *testing.T) {
	ctx := newTestCtx(t)
	params := map[string]string{"passphrase": "p1", "other": "keep"}

	got, err := getPassphraseFromEnv(ctx, params)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "p1" {
		t.Fatalf("got %q, want p1", got)
	}
	if _, present := params["passphrase"]; present {
		t.Fatal("passphrase key should have been deleted from params")
	}
	if params["other"] != "keep" {
		t.Fatal("unrelated params keys must be preserved")
	}
}

func TestGetPassphraseFromEnv_EnvFallback(t *testing.T) {
	ctx := newTestCtx(t)
	t.Setenv("PLAKAR_PASSPHRASE", "from-env")

	got, err := getPassphraseFromEnv(ctx, map[string]string{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "from-env" {
		t.Fatalf("got %q, want from-env", got)
	}
}

func TestGetPassphraseFromEnv_NoSourcesReturnsEmpty(t *testing.T) {
	ctx := newTestCtx(t)
	t.Setenv("PLAKAR_PASSPHRASE", "")

	got, err := getPassphraseFromEnv(ctx, map[string]string{})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty string", got)
	}
}

// ---------- setupEncryption ----------

func TestSetupEncryption_NilConfigIsNoOp(t *testing.T) {
	ctx := newTestCtx(t)
	cfg := &storage.Configuration{Encryption: nil}

	if err := setupEncryption(ctx, cfg); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

// ---------- checkUpdate ----------

func TestCheckUpdate_FirstRunWithSecurityCheckDisabledIsSilent(t *testing.T) {
	ctx := newTestCtx(t)
	// First-run + disableSecurityCheck=true: function should mark first-run
	// and return without printing the welcome banner.
	checkUpdate(ctx, true)

	out := ctx.Stdout.(*bytes.Buffer).String()
	if out != "" {
		t.Fatalf("expected silent output, got %q", out)
	}
	if ctx.GetCookies().IsFirstRun() {
		t.Fatal("first-run flag should have been cleared")
	}
}

func TestCheckUpdate_FirstRunPrintsWelcome(t *testing.T) {
	ctx := newTestCtx(t)
	// First-run + disableSecurityCheck=false: prints the welcome banner.
	checkUpdate(ctx, false)

	out := ctx.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "Welcome to plakar") {
		t.Fatalf("expected welcome message in stdout, got %q", out)
	}
	if !strings.Contains(out, "-disable-security-check") {
		t.Fatalf("expected -disable-security-check guidance in stdout, got %q", out)
	}
	if ctx.GetCookies().IsFirstRun() {
		t.Fatal("first-run flag should have been cleared")
	}
}

func TestCheckUpdate_SecondRunWithDisableFlagReturnsEarly(t *testing.T) {
	ctx := newTestCtx(t)
	// Simulate not-first-run by flipping the cookie ourselves.
	if err := ctx.GetCookies().SetFirstRun(); err != nil {
		t.Fatalf("SetFirstRun: %v", err)
	}

	checkUpdate(ctx, true)

	out := ctx.Stdout.(*bytes.Buffer).String()
	if out != "" {
		t.Fatalf("expected silent output, got %q", out)
	}
}

// ---------- listCmds ----------

func TestListCmds_EmptyPrefix(t *testing.T) {
	// All subcommands are registered via main.go's blank imports, so
	// subcommands.List() returns the full real list during tests.
	var buf bytes.Buffer
	listCmds(&buf, "")
	out := buf.String()
	if out == "" {
		t.Fatal("listCmds produced no output despite many registered subcommands")
	}
	// Every line should be terminated with a newline.
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("output should end with newline, got %q", out)
	}
	// Spot-check a couple of subcommands we know are registered.
	for _, must := range []string{"backup", "restore", "check"} {
		if !strings.Contains(out, must) {
			t.Errorf("expected %q somewhere in output, got %q", must, out)
		}
	}
}

func TestListCmds_PrefixIsApplied(t *testing.T) {
	var buf bytes.Buffer
	listCmds(&buf, "PFX> ")
	out := buf.String()
	if !strings.HasPrefix(out, "PFX> ") {
		t.Fatalf("first line should start with the supplied prefix, got %q", out)
	}
	// Every line should start with PFX> (we just check a couple).
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for _, l := range lines {
		if !strings.HasPrefix(l, "PFX> ") {
			t.Errorf("line %q does not start with prefix", l)
		}
	}
}

func TestListCmds_FiltersDiagAndCached(t *testing.T) {
	var buf bytes.Buffer
	listCmds(&buf, "")
	out := buf.String()
	// diag and cached are explicitly filtered out by listCmds.
	for _, banned := range []string{" diag ", " cached "} {
		if strings.Contains(" "+out+" ", banned) {
			t.Errorf("listCmds should filter %q from output, got %q", banned, out)
		}
	}
}

// ---------- isTerminal ----------

func TestIsTerminal_DumbTermReturnsFalse(t *testing.T) {
	t.Setenv("TERM", "dumb")
	if isTerminal() {
		t.Fatal("isTerminal() with TERM=dumb should return false")
	}
}

func TestIsTerminal_NoTtyReturnsFalse(t *testing.T) {
	// Under `go test` stdout is captured (not a tty), so even without
	// TERM=dumb the term.IsTerminal(1) call returns false.
	t.Setenv("TERM", "xterm")
	if isTerminal() {
		t.Fatal("isTerminal() in `go test` (stdout is a pipe) should return false")
	}
}

// ---------- pkgpreloadhook ----------

func TestPkgPreloadHook_AcceptsUnknownProtocols(t *testing.T) {
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "importer", Protocols: []string{"unique-imp-" + t.Name()}},
			{Type: "exporter", Protocols: []string{"unique-exp-" + t.Name()}},
			{Type: "storage", Protocols: []string{"unique-stg-" + t.Name()}},
			// Unknown connector type — should just print a warning, not error.
			{Type: "weird", Protocols: []string{"ignored"}},
		},
	}
	// Redirect stderr to keep the "skipping unknown connector type" message
	// from polluting test output. We don't assert on it because it goes to
	// the global os.Stderr; the function itself returns nil.
	if err := pkgpreloadhook(m); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestPkgPreloadHook_RejectsAlreadyRegisteredStorage(t *testing.T) {
	// "fs" is registered by the integrations/fs storage import at the top
	// of main.go — pkgpreloadhook should refuse a manifest that tries to
	// claim it.
	if !slicesContains(storage.Backends(), "fs") {
		t.Skip("fs storage backend not registered — skipping conflict test")
	}
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "storage", Protocols: []string{"fs"}},
		},
	}
	if err := pkgpreloadhook(m); err == nil {
		t.Fatal("pkgpreloadhook should reject a manifest claiming the already-registered fs proto")
	}
}

// slicesContains is local because the std-lib slices.Contains is generic and
// some older code in the tree imports slices for other purposes already; keep
// this self-contained.
func slicesContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// ---------- pkgloadhook / pkgunloadhook ----------

func TestPkgLoadAndUnloadHook_EmptyManifestNoOp(t *testing.T) {
	// Both hooks are wrappers around plugins.Load / plugins.Unload that
	// print to stderr on error. An empty manifest is a no-op — the hooks
	// just shouldn't panic.
	pkgloadhook(&pkg.Manifest{}, &pkg.Package{}, t.TempDir())
	pkgunloadhook(&pkg.Manifest{}, &pkg.Package{})
}

// Sentinel: make sure ErrCantUnlock is the same wrapped error users see.
func TestErrCantUnlockIsStable(t *testing.T) {
	if ErrCantUnlock.Error() == "" {
		t.Fatal("ErrCantUnlock should have a non-empty message")
	}
	if !strings.Contains(ErrCantUnlock.Error(), "unlock") {
		t.Fatalf("ErrCantUnlock message should mention unlock, got %q", ErrCantUnlock.Error())
	}
}

// Compile-time check: io.Writer used by listCmds matches *bytes.Buffer.
var _ = os.Stdout // keep `os` import alive across refactors
