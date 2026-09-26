package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/config"
	pkgcmd "github.com/PlakarKorp/plakar/subcommands/pkg"
)

func TestSetupPkgManagerRegistries(t *testing.T) {
	ctx := newTestCtx(t)
	ctx.ConfigDir = t.TempDir()
	ctx.Config = config.NewConfig()
	ctx.Config.Registries["lab"] = config.RegistryConfig{URL: "https://lab.example.org/"}

	if err := setupPkgManager(ctx, ctx.ConfigDir, t.TempDir(), t.TempDir()); err != nil {
		t.Fatalf("setupPkgManager: %v", err)
	}
	if ctx.GetPkgManager() == nil {
		t.Fatal("no package manager set")
	}
	if s := ctx.Stderr.(*bytes.Buffer).String(); s != "" {
		t.Fatalf("unexpected stderr %q", s)
	}
}

// A registry the package manager refuses shows the configuration is passed
// to it. It must not be fatal, so that it can be removed.
func TestSetupPkgManagerBadRegistry(t *testing.T) {
	ctx := newTestCtx(t)
	ctx.ConfigDir = t.TempDir()
	ctx.Config = config.NewConfig()
	ctx.Config.Registries["bad"] = config.RegistryConfig{URL: "ftp://bad.example.org/"}
	if err := config.Save(ctx.ConfigDir, ctx.Config); err != nil {
		t.Fatal(err)
	}

	if err := setupPkgManager(ctx, ctx.ConfigDir, t.TempDir(), t.TempDir()); err != nil {
		t.Fatalf("setupPkgManager: %v", err)
	}
	if ctx.GetPkgManager() == nil {
		t.Fatal("no package manager set")
	}
	want := progName() + ": " + filepath.Join(ctx.ConfigDir, "registries.yml") + ": "
	stderr := ctx.Stderr.(*bytes.Buffer).String()
	if !strings.HasPrefix(stderr, want) || !strings.Contains(stderr, `"bad"`) ||
		!strings.HasSuffix(stderr, "; ignoring additional registries\n") {
		t.Fatalf("stderr = %q", stderr)
	}

	cmd := &pkgcmd.PkgRegistryRm{}
	if err := cmd.Parse(ctx, []string{"bad"}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := cmd.Execute(ctx, nil); err != nil {
		t.Fatalf("pkg registry rm bad: %v", err)
	}
	cfg, err := config.Load(ctx.ConfigDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Registries) != 0 {
		t.Fatalf("registries = %v, want none", cfg.Registries)
	}
}
