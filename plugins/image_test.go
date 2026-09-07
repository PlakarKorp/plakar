package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PlakarKorp/pkg"
)

// fakeImageDocker prepends a docker stand-in to PATH that emulates
// `image inspect` and `pull` against a state dir of present-<ref> files.
// A pull makes its ref present with the given pullID, or fails when pullID
// is empty.
func fakeImageDocker(t *testing.T, pullID string) (statedir string) {
	t.Helper()
	statedir = t.TempDir()

	pull := `echo "pull access denied" >&2; exit 1`
	if pullID != "" {
		pull = fmt.Sprintf(`key=$(printf %%s "$2" | tr '/:' '__')
	printf %%s %s > "$STATE/present-$key"
	exit 0`, pullID)
	}

	script := fmt.Sprintf(`#!/bin/sh
STATE=%s
echo "$@" >> "$STATE/log"
case "$1" in
image)
	# image inspect --format {{.Id}} <ref>
	key=$(printf %%s "$5" | tr '/:' '__')
	if [ -f "$STATE/present-$key" ]; then cat "$STATE/present-$key"; exit 0; fi
	echo "Error: No such image: $5" >&2
	exit 1;;
pull)
	%s;;
esac
exit 1
`, statedir, pull)

	bindir := filepath.Join(statedir, "bin")
	if err := os.Mkdir(bindir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bindir, "docker"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bindir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return statedir
}

func markPresent(t *testing.T, statedir, ref, id string) {
	t.Helper()
	key := strings.NewReplacer("/", "_", ":", "_").Replace(ref)
	if err := os.WriteFile(filepath.Join(statedir, "present-"+key), []byte(id), 0644); err != nil {
		t.Fatal(err)
	}
}

func dockerLog(t *testing.T, statedir string) string {
	t.Helper()
	log, err := os.ReadFile(filepath.Join(statedir, "log"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(log)
}

const imageManifestID = "sha256:1111"

func imageManifest(image string) *pkg.Manifest {
	return &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "importer", Protocols: []string{"x"}, ImageID: imageManifestID, Image: image},
		},
	}
}

func TestEnsureImagesNativeManifestUntouched(t *testing.T) {
	statedir := fakeImageDocker(t, "")

	m := &pkg.Manifest{Connectors: []pkg.ManifestConnector{
		{Type: "importer", Protocols: []string{"x"}, Executable: "noop"},
	}}
	if err := EnsureImages(context.Background(), m, io.Discard, io.Discard); err != nil {
		t.Fatalf("EnsureImages: %v", err)
	}
	if log := dockerLog(t, statedir); log != "" {
		t.Fatalf("native manifest should not touch docker, got: %s", log)
	}
}

func TestEnsureImagesAlreadyPresent(t *testing.T) {
	statedir := fakeImageDocker(t, "")
	markPresent(t, statedir, imageManifestID, imageManifestID)

	if err := EnsureImages(context.Background(), imageManifest("reg/img:1"), io.Discard, io.Discard); err != nil {
		t.Fatalf("EnsureImages: %v", err)
	}
	if log := dockerLog(t, statedir); strings.Contains(log, "pull") {
		t.Fatalf("present image should not be pulled, got: %s", log)
	}
}

func TestEnsureImagesAbsentWithoutRef(t *testing.T) {
	fakeImageDocker(t, "")

	err := EnsureImages(context.Background(), imageManifest(""), io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "rebuild the package") {
		t.Fatalf("want a rebuild error, got: %v", err)
	}
}

func TestEnsureImagesPullMatches(t *testing.T) {
	statedir := fakeImageDocker(t, imageManifestID)

	if err := EnsureImages(context.Background(), imageManifest("reg/img:1"), io.Discard, io.Discard); err != nil {
		t.Fatalf("EnsureImages: %v", err)
	}
	if log := dockerLog(t, statedir); !strings.Contains(log, "pull reg/img:1") {
		t.Fatalf("expected a pull, got: %s", log)
	}
}

func TestEnsureImagesPullMismatch(t *testing.T) {
	fakeImageDocker(t, "sha256:2222")

	err := EnsureImages(context.Background(), imageManifest("reg/img:1"), io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "refusing to install") {
		t.Fatalf("want an ID-mismatch error, got: %v", err)
	}
}

func TestEnsureImagesPullFails(t *testing.T) {
	fakeImageDocker(t, "")

	err := EnsureImages(context.Background(), imageManifest("reg/img:1"), io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "failed to pull") {
		t.Fatalf("want a pull error, got: %v", err)
	}
}
