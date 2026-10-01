package plugins

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/PlakarKorp/pkg"
)

// EnsureImages makes sure the images pinned by a manifest are present in the
// local docker store: an image already present by ID is used as-is (the
// self-built case), a missing one is pulled from its registry reference and
// the pulled ID must match the pin. Pull progress is streamed to stdout and
// stderr.
func EnsureImages(ctx context.Context, m *pkg.Manifest, stdout, stderr io.Writer) error {
	for _, conn := range m.Connectors {
		if conn.ImageID == "" {
			continue
		}

		if _, err := imageID(ctx, conn.ImageID); err == nil {
			continue
		}

		if conn.Image == "" {
			return fmt.Errorf("image %s is not present; rebuild the package with plakar pkg create -container",
				conn.ImageID)
		}

		pull := exec.CommandContext(ctx, "docker", "pull", conn.Image)
		pull.Stdout = stdout
		pull.Stderr = stderr
		if err := pull.Run(); err != nil {
			return fmt.Errorf("failed to pull %s: %w", conn.Image, err)
		}

		id, err := imageID(ctx, conn.Image)
		if err != nil {
			return err
		}
		if id != conn.ImageID {
			return fmt.Errorf("pulled image %s has ID %s, but the package pins %s: refusing to install",
				conn.Image, id, conn.ImageID)
		}
	}
	return nil
}

func imageID(ctx context.Context, ref string) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "image", "inspect", "--format", "{{.Id}}", ref).Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect image %s: %w", ref, err)
	}
	return strings.TrimSpace(string(out)), nil
}
