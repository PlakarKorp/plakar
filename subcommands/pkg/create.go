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
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"golang.org/x/mod/semver"
)

type PkgCreate struct {
	subcommands.SubcommandBase

	Base         string
	Out          string
	Manifest     pkg.Manifest
	ManifestPath string
}

func (cmd *PkgCreate) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg create", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [-out plugin] manifest.yaml version\n",
			flags.Name())
	}

	flags.StringVar(&cmd.Out, "out", "", "Plugin file to create")
	flags.Parse(args)

	if flags.NArg() != 2 {
		return fmt.Errorf("wrong usage")
	}

	var (
		manifest = flags.Arg(0)
		version  = flags.Arg(1)
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

	if cmd.Out == "" {
		p := fmt.Sprintf("%s_%s_%s_%s.ptar", cmd.Manifest.Name, version, GOOS, GOARCH)
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

	source, err := snapshot.NewSource(ctx, 0, imp)
	if err != nil {
		return 1, err
	}

	backupOptions := &snapshot.BuilderOptions{
		NoCheckpoint: true,
		NoCommit:     true,
	}

	snap, err := snapshot.CreateWithRepositoryWriter(repoWriter, backupOptions, objects.NilMac)
	if err != nil {
		return 1, fmt.Errorf("failed to create snapshot: %w", err)
	}

	err = snap.Backup(source)
	if err != nil {
		return 1, fmt.Errorf("failed to populate the snapshot: %w", err)
	}

	_, err = snap.PutSnapshot()
	if err != nil {
		return 1, fmt.Errorf("failed to commit snapshot: %w", err)
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
