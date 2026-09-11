/*
 * Copyright (c) 2025 Matthieu Masson <matthieu.masson@plakar.io>
 * Copyright (c) 2025 Omar Polo <omar.polo@plakar.io>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package pkg

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
	"golang.org/x/mod/semver"
)

type PkgCreate struct {
	subcommands.SubcommandBase

	Base         string
	Out          string
	Manifest     pkg.Manifest
	ManifestPath string
	Version      string
	Arch         string
	Container    bool
	ImageRef     string
}

func (cmd *PkgCreate) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "pkg create",
	}
	c.Flags().StringVar(&cmd.Out, "out", "", "Plugin file to create")
	c.Flags().BoolVar(&cmd.Container, "container", false, "Build a container image from the Dockerfile next to the manifest and package a reference to it instead of executables")
	c.Flags().StringVar(&cmd.ImageRef, "image-ref", "", "Registry reference to record as the image pull source (container packages only)")
	return c
}

func (cmd *PkgCreate) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) != 2 {
		return fmt.Errorf("wrong usage")
	}

	var (
		manifest = rest[0]
		version  = rest[1]
	)

	if !semver.IsValid(version) {
		return fmt.Errorf("bad version string: %s", version)
	}

	if !filepath.IsAbs(manifest) {
		manifest = filepath.Join(ctx.CWD, manifest)
	} else {
		manifest = filepath.Clean(manifest)
	}
	cmd.Base = filepath.Dir(manifest)
	cmd.ManifestPath = manifest

	if filepath.Base(manifest) != "manifest.yaml" {
		return fmt.Errorf("manifest's file name must be manifest.yaml")
	}

	fp, err := os.Open(manifest)
	if err != nil {
		return fmt.Errorf("can't open %s: %w", manifest, err)
	}
	defer fp.Close()

	if err := cmd.Manifest.Parse(fp); err != nil {
		return fmt.Errorf("failed to parse the manifest %s: %w", manifest, err)
	}

	GOOS := runtime.GOOS
	GOARCH := runtime.GOARCH
	if goosEnv := os.Getenv("GOOS"); goosEnv != "" {
		GOOS = goosEnv
	}
	if goarchEnv := os.Getenv("GOARCH"); goarchEnv != "" {
		GOARCH = goarchEnv
	}
	cmd.Version = version
	cmd.Arch = GOARCH

	if cmd.Container && GOOS != "linux" {
		return fmt.Errorf("container packages are linux-only")
	}
	if cmd.ImageRef != "" && !cmd.Container {
		return fmt.Errorf("-image-ref requires -container")
	}

	if cmd.Out == "" {
		p := fmt.Sprintf("%s_%s_%s_%s.ptar", cmd.Manifest.Name, version, GOOS, GOARCH)
		if cmd.Container {
			p = fmt.Sprintf("%s_%s_%s_%s.ptar", cmd.Manifest.Name, version, pkg.OSContainer, GOARCH)
		}
		cmd.Out = filepath.Join(ctx.CWD, p)
	}

	return nil
}

// containerize builds the image from the Dockerfile next to the manifest and
// rewrites the connectors to carry the resulting image ID instead of their
// executable, which becomes the command run inside the container.
func (cmd *PkgCreate) containerize(ctx *appcontext.AppContext) error {
	if _, err := os.Stat(filepath.Join(cmd.Base, "Dockerfile")); err != nil {
		return fmt.Errorf("container packages need a Dockerfile next to the manifest: %w", err)
	}

	// docker build writes the image id to a file
	iid, err := os.CreateTemp("", "plakar-iidfile-")
	if err != nil {
		return err
	}
	iid.Close()
	defer os.Remove(iid.Name())

	// We only reference ImageIDs internally, but we still want to produce a
	// human readable tag so that docker images stays nice.
	tag := fmt.Sprintf("plakar/plugin-%s:%s", cmd.Manifest.Name, cmd.Version)
	build := exec.CommandContext(ctx, "docker", "build",
		"-t", tag,
		"--platform", "linux/"+cmd.Arch,
		"--iidfile", iid.Name(),
		cmd.Base)
	build.Stdout = ctx.Stdout
	build.Stderr = ctx.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	rawID, err := os.ReadFile(iid.Name())
	if err != nil {
		return err
	}
	imageID := strings.TrimSpace(string(rawID))
	if imageID == "" {
		return fmt.Errorf("docker build produced no image ID")
	}

	for i := range cmd.Manifest.Connectors {
		conn := &cmd.Manifest.Connectors[i]
		conn.Args = append([]string{conn.Executable}, conn.Args...)
		conn.Executable = ""
		conn.ImageID = imageID
		conn.Image = cmd.ImageRef
	}
	return nil
}

func (cmd *PkgCreate) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	if cmd.Container {
		if err := cmd.containerize(ctx); err != nil {
			return 1, err
		}
	}

	for i := range cmd.Manifest.Connectors {
		if err := cmd.Manifest.Connectors[i].Validate(); err != nil {
			return 1, fmt.Errorf("invalid connector: %w", err)
		}
	}

	// When in container mode the manifest is built from the Manifest struct in
	// memory and contains ImageID and possibly Image.
	// Otherwise when using native mode we just read the file from disk.
	var manifestData []byte
	var err error
	if cmd.Container {
		manifestData, err = yaml.Marshal(&cmd.Manifest)
	} else {
		manifestData, err = os.ReadFile(cmd.ManifestPath)
	}
	if err != nil {
		return 1, fmt.Errorf("failed to load the manifest: %w", err)
	}

	storageConfiguration := storage.NewConfiguration()
	storageConfiguration.Encryption = nil
	storageConfiguration.Packfile.MaxSize = math.MaxUint64
	hasher := hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)

	serializedConfig, err := storageConfiguration.ToBytes()
	if err != nil {
		return 1, fmt.Errorf("failed to serialize configuration: %w", err)
	}

	rd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serializedConfig))
	if err != nil {
		return 1, fmt.Errorf("failed to wrap configuration: %w", err)
	}
	wrappedConfig, err := io.ReadAll(rd)
	if err != nil {
		return 1, fmt.Errorf("failed to read wrapped configuration: %w", err)
	}

	st, err := storage.Create(ctx.GetInner(), map[string]string{
		"location": "ptar:" + cmd.Out,
	}, wrappedConfig)
	if err != nil {
		return 1, fmt.Errorf("failed to create the storage: %w", err)
	}

	repo, err := repository.New(ctx.GetInner(), nil, st, wrappedConfig)
	if err != nil {
		return 1, fmt.Errorf("failed to create ptar: %w", err)
	}

	imp := &pkgerImporter{
		manifestData: manifestData,
		manifest:     &cmd.Manifest,
		cwd:          cmd.Base,
	}
	source, err := snapshot.NewSource(ctx, imp)
	if err != nil {
		return 1, err
	}

	backupOptions := &snapshot.BuilderOptions{
		NoCheckpoint: true,
	}

	snap, err := snapshot.Create(repo, repository.PtarType, "", objects.NilMac, backupOptions)
	if err != nil {
		return 1, fmt.Errorf("failed to create snapshot: %w", err)
	}
	defer snap.Close()

	if err = snap.Backup(source); err != nil {
		return 1, fmt.Errorf("failed to populate the snapshot: %w", err)
	}

	if err = snap.Commit(); err != nil {
		return 1, fmt.Errorf("failed to commit snapshot: %w", err)
	}

	if err := st.Close(ctx); err != nil {
		return 1, fmt.Errorf("failed to close the storage: %w", err)
	}

	if snap.Header.GetSource(0).Summary.Below.Errors != 0 {
		return 1, fmt.Errorf("failed to package all the files")
	}

	fmt.Fprintf(ctx.Stdout, "Plugin created successfully: %s\n", cmd.Out)
	return 0, nil
}
