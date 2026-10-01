//go:build linux
package pkg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	ppkg "github.com/PlakarKorp/pkg"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

const testImageID = "sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

// fakeContainerCLI prepends a docker stand-in to PATH that records its
// invocations and writes a fixed image ID to the --iidfile.
func fakeContainerCLI(t *testing.T) (argsFile string) {
	t.Helper()
	dir := t.TempDir()
	argsFile = filepath.Join(dir, "args")

	script := fmt.Sprintf(`#!/bin/sh
echo "$@" >> %s
case "$1" in
build)
	while [ $# -gt 1 ]; do
		if [ "$1" = "--iidfile" ]; then echo %s > "$2"; fi
		shift
	done
	exit 0;;
tag)
	exit 0;;
esac
exit 1
`, argsFile, testImageID)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argsFile
}

func writeContainerPackage(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(
		"name: myplugin\nconnectors:\n  - type: importer\n    protocols:\n      - myproto\n    executable: myplugin\n    args:\n      - --verbose\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(
		"FROM scratch\nCOPY myplugin /usr/local/bin/\n"), 0644))
}

func TestPkgCreateContainerParse(t *testing.T) {
	ctx := newCtx(t)
	writeContainerPackage(t, ctx.CWD)

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-container", "manifest.yaml", "v1.2.3"}))
	require.True(t, cmd.Container)

	// Container packages are linux-flavored regardless of the host OS.
	want := fmt.Sprintf("myplugin_v1.2.3_oci_%s.ptar", runtime.GOARCH)
	require.Equal(t, want, filepath.Base(cmd.Out))

	// -image-ref only makes sense for container packages.
	err := (&PkgCreate{}).Parse(ctx, []string{"-image-ref", "reg/img:1", "manifest.yaml", "v1.2.3"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "-image-ref requires -container")
}

func TestPkgCreateContainerizeStampsConnectors(t *testing.T) {
	fakeContainerCLI(t)
	ctx := newCtx(t)
	writeContainerPackage(t, ctx.CWD)

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-container", "-image-ref", "reg/plugin-myplugin:v1.2.3", "manifest.yaml", "v1.2.3"}))
	require.NoError(t, cmd.containerize(ctx))

	conn := cmd.Manifest.Connectors[0]
	require.Equal(t, testImageID, conn.ImageID)
	require.Equal(t, "reg/plugin-myplugin:v1.2.3", conn.Image)
	require.Empty(t, conn.Executable)
	// The executable becomes the command run inside the container.
	require.Equal(t, []string{"myplugin", "--verbose"}, conn.Args)
	require.NoError(t, conn.Validate())
}

func TestPkgCreateContainerizeNeedsDockerfile(t *testing.T) {
	fakeContainerCLI(t)
	ctx := newCtx(t)
	writeContainerPackage(t, ctx.CWD)
	require.NoError(t, os.Remove(filepath.Join(ctx.CWD, "Dockerfile")))

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-container", "manifest.yaml", "v1.2.3"}))
	err := cmd.containerize(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Dockerfile")
}

func TestPkgCreateContainerExecuteBuildsPtar(t *testing.T) {
	argsFile := fakeContainerCLI(t)

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	work := t.TempDir()
	ctx.CWD = work
	writeContainerPackage(t, work)
	// No executable on disk: the binary lives in the image only.

	out := filepath.Join(work, "out.ptar")
	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-container", "-out", out, "manifest.yaml", "v1.2.3"}))

	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	info, err := os.Stat(out)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
	require.Contains(t, bufOut.String(), "Plugin created successfully")

	// The build was invoked with an explicit platform and the cosmetic tag
	// folded into it.
	invocations, err := os.ReadFile(argsFile)
	require.NoError(t, err)
	require.Contains(t, string(invocations), "--platform linux/"+runtime.GOARCH)
	require.Contains(t, string(invocations), "-t plakar/plugin-myplugin:v1.2.3")
}

func TestPkgCreateNativeRejectsInvalidConnector(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	work := t.TempDir()
	ctx.CWD = work
	// A source manifest carrying an image pin is invalid: pins are stamped
	// at create time, never hand-written next to an executable.
	require.NoError(t, os.WriteFile(filepath.Join(work, "manifest.yaml"), []byte(
		"name: myplugin\nconnectors:\n  - type: importer\n    executable: myplugin\n    image_id: sha256:aa\n"), 0644))

	cmd := &PkgCreate{}
	err := cmd.Parse(ctx, []string{"-out", filepath.Join(work, "out.ptar"), "manifest.yaml", "v1.2.3"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "exactly one of")
}

func TestPkgAddContainerRunsInstallHook(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("container packages are linux-only")
	}
	fakeContainerCLI(t)

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	work := t.TempDir()
	ctx.CWD = work
	writeContainerPackage(t, work)

	cmd := &PkgCreate{}
	require.NoError(t, cmd.Parse(ctx, []string{"-container", "manifest.yaml", "v1.2.3"}))
	status, err := cmd.Execute(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	newManager := func(hook func(*ppkg.Manifest) error) *ppkg.Manager {
		base := t.TempDir()
		backend, err := ppkg.NewFlatBackend(ctx.GetInner(),
			filepath.Join(base, "plugins"), filepath.Join(base, "cache"),
			&ppkg.FlatBackendOptions{InstallHook: hook})
		require.NoError(t, err)
		mgr, err := ppkg.New(backend, nil)
		require.NoError(t, err)
		return mgr
	}

	// The install hook sees the stamped manifest; a container-flavored
	// package installs from the local file.
	var seen []*ppkg.Manifest
	mgr := newManager(func(m *ppkg.Manifest) error { seen = append(seen, m); return nil })
	require.NoError(t, mgr.Add(cmd.Out, nil))
	require.Len(t, seen, 1)
	require.Equal(t, testImageID, seen[0].Connectors[0].ImageID)

	var installed []*ppkg.Package
	for p, err := range mgr.List() {
		require.NoError(t, err)
		installed = append(installed, p)
	}
	require.Len(t, installed, 1)
	require.Equal(t, ppkg.OSContainer, installed[0].OperatingSystem)

	// A failing hook aborts the install.
	mgr = newManager(func(m *ppkg.Manifest) error { return fmt.Errorf("image not present") })
	err = mgr.Add(cmd.Out, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "image not present")
	for range mgr.List() {
		t.Fatal("nothing should be installed after a failed install hook")
	}
}

func TestPkgerImporterManifestData(t *testing.T) {
	dir := t.TempDir()
	data := []byte("name: myplugin\nconnectors:\n  - type: importer\n    image_id: sha256:aa\n")

	var m ppkg.Manifest
	require.NoError(t, m.Parse(bytes.NewReader(data)))
	imp := &pkgerImporter{cwd: dir, manifest: &m, manifestData: data}

	ch := make(chan *connectors.Record, 16)
	require.NoError(t, imp.Import(context.Background(), ch, nil))

	var recs []*connectors.Record
	for r := range ch {
		recs = append(recs, r)
	}
	// Only the in-memory manifest: no executable to package.
	require.Len(t, recs, 1)
	require.Equal(t, "/manifest.yaml", recs[0].Pathname)

	content, err := io.ReadAll(recs[0].Reader)
	require.NoError(t, err)
	require.Equal(t, data, content)
}
