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
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"

	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/subcommands"
)

type PkgCreate struct {
	subcommands.SubcommandBase

	Base         string
	Out          string
	Manifest     plugins.Manifest
	ManifestPath string
}

func (cmd *PkgCreate) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg create", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [-out plugin] manifest.yaml\n",
			flags.Name())
	}

	flags.StringVar(&cmd.Out, "out", "", "Plugin file to create")
	flags.Parse(args)

	if flags.NArg() != 1 {
		return fmt.Errorf("wrong usage")
	}

	manifest := flags.Arg(0)
	if !filepath.IsAbs(manifest) {
		manifest = filepath.Join(ctx.CWD, manifest)
	} else {
		manifest = filepath.Clean(manifest)
	}
	cmd.Base = filepath.Dir(manifest)
	cmd.ManifestPath = manifest

	if err := plugins.ParseManifestFile(manifest, &cmd.Manifest); err != nil {
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

	if cmd.Out == "" {
		p := fmt.Sprintf("%s_%s_%s_%s.ptar", cmd.Manifest.Name, cmd.Manifest.Version, GOOS, GOARCH)
		cmd.Out = filepath.Join(ctx.CWD, p)
	}

	return nil
}

func (cmd *PkgCreate) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
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

	identifier := objects.RandomMAC()
	scanCache, err := repo.AppContext().GetCache().Scan(identifier)
	if err != nil {
		return 1, fmt.Errorf("failed to get the scan cache: %w", err)
	}

	repoWriter := repo.NewRepositoryWriter(scanCache, identifier, repository.PtarType, "")
	imp := &pkgerImporter{
		manifestPath: cmd.ManifestPath,
		manifest:     &cmd.Manifest,
		cwd:          cmd.Base,
	}

	snap, err := snapshot.CreateWithRepositoryWriter(repoWriter)
	if err != nil {
		return 1, fmt.Errorf("failed to create snapshot: %w", err)
	}

	backupOptions := &snapshot.BackupOptions{
		NoCheckpoint: true,
		NoCommit:     true,
	}

	err = snap.Backup(imp, backupOptions)

	if err != nil {
		return 1, fmt.Errorf("failed to populate the snapshot: %w", err)
	}

	// We are done with everything we can now stop the backup routines.
	repoWriter.PackerManager.Wait()
	err = repoWriter.CommitTransaction(identifier)
	if err != nil {
		return 1, fmt.Errorf("failed to commit transaction: %w", err)
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
