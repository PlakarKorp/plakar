/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION
 * OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN
 * CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package ptar

import (
	"bytes"
	"flag"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/PlakarKorp/kloset/compression"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"
)

func init() {
	subcommands.Register(Ptar, subcommands.BeforeRepositoryWithStorage, "ptar")
}

type listFlag []string

func (l *listFlag) String() string {
	return fmt.Sprint([]string(*l))
}

func (l *listFlag) Set(value string) error {
	if slices.Contains(*l, value) {
		return nil
	}
	*l = append(*l, value)
	return nil
}

func Ptar(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		klosetPath string
		klosetUUID = uuid.Must(uuid.NewRandom())

		//AllowWeak     bool
		algo          string
		plaintext     bool
		nocompression bool
		overwrite     bool

		syncTargets   listFlag
		syncSecrets   [][]byte
		backupTargets listFlag

		repoSecret []byte
	)

	flags := flag.NewFlagSet("ptar", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: plakar %s [OPTIONS] -o out.ptar [@location | path]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&algo, "hashing", hashing.DEFAULT_HASHING_ALGORITHM, "hashing algorithm to use for digests")
	flags.BoolVar(&plaintext, "plaintext", false, "disable transparent encryption")
	flags.BoolVar(&nocompression, "no-compression", false, "disable transparent compression")
	flags.BoolVar(&overwrite, "overwrite", false, "overwrite the ptar archive if it already exists")
	flags.Var(&syncTargets, "k", "add a kloset location to include in the ptar archive (can be specified multiple times)")
	flags.Var(&syncTargets, "kloset", "add a kloset location to include in the ptar archive (can be specified multiple times)")
	flags.StringVar(&klosetPath, "o", "", "name of the ptar archive to create")
	flags.Parse(args)

	if klosetPath == "" {
		return fmt.Errorf("%s: -o option must be specified", flag.CommandLine.Name())
	}

	if len(syncTargets) == 0 && flags.NArg() == 0 {
		backupTargets = []string{ctx.CWD}
	}

	if len(flags.Args()) > 0 {
		backupTargets = make([]string, len(flags.Args()))
		copy(backupTargets, flags.Args())
	}

	for _, syncTarget := range syncTargets {
		var peerSecret []byte

		storeConfig, err := ctx.Config.GetRepository(syncTarget)
		if err != nil {
			return fmt.Errorf("peer repository: %w", err)
		}

		peerStore, peerStoreSerializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
		if err != nil {
			return err
		}

		peerStoreConfig, err := storage.NewConfigurationFromWrappedBytes(peerStoreSerializedConfig)
		if err != nil {
			return err
		}

		if peerStoreConfig.Encryption != nil {
			if pass, ok := storeConfig["passphrase"]; ok {
				key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, []byte(pass))
				if err != nil {
					return err
				}
				if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
					return fmt.Errorf("invalid passphrase")
				}
				peerSecret = key
			} else {
				for {
					passphrase, err := utils.GetPassphrase("source repository")
					if err != nil {
						fmt.Fprintf(os.Stderr, "%s\n", err)
						continue
					}

					key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, passphrase)
					if err != nil {
						return err
					}
					if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
						return fmt.Errorf("invalid passphrase")
					}
					peerSecret = key
					break
				}
			}
		}

		peerCtx := appcontext.NewAppContextFrom(ctx)
		_, err = repository.NewNoRebuild(peerCtx.GetInner(), peerSecret, peerStore, peerStoreSerializedConfig, true)
		if err != nil {
			return err
		}
		syncSecrets = append(syncSecrets, peerSecret)
	}

	if hashing.GetHasher(strings.ToUpper(algo)) == nil {
		return fmt.Errorf("%s: unknown hashing algorithm", flag.CommandLine.Name())
	}

	if !plaintext {
		var passphrase []byte

		envPassphrase, ok := os.LookupEnv("PLAKAR_PASSPHRASE")
		if ctx.KeyFromFile == "" {
			if ok {
				passphrase = []byte(envPassphrase)
			} else {
				tmp, err := utils.GetPassphraseConfirm("repository", 0., 3)
				if err != nil {
					return err
				}
				passphrase = tmp
			}
		} else {
			passphrase = []byte(ctx.KeyFromFile)
		}

		if len(passphrase) == 0 {
			return fmt.Errorf("can't encrypt the repository with an empty passphrase")
		}

		repoSecret = passphrase
	}

	storageConfiguration := storage.NewConfiguration()
	storageConfiguration.RepositoryID = klosetUUID

	if nocompression {
		storageConfiguration.Compression = nil
	} else {
		storageConfiguration.Compression = compression.NewDefaultConfiguration()
	}

	hashingConfiguration, err := hashing.LookupDefaultConfiguration(strings.ToUpper(algo))
	if err != nil {
		return err
	}
	storageConfiguration.Hashing = *hashingConfiguration

	var hasher hash.Hash
	var key []byte
	if !plaintext {
		storageConfiguration.Encryption = encryption.NewDefaultConfiguration()

		key, err = encryption.DeriveKey(storageConfiguration.Encryption.KDFParams,
			repoSecret)
		if err != nil {
			return err
		}

		canary, err := encryption.DeriveCanary(storageConfiguration.Encryption, key)
		if err != nil {
			return err
		}
		storageConfiguration.Encryption.Canary = canary
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, key)
		//ctx.SetSecret(key)
	} else {
		storageConfiguration.Encryption = nil
		hasher = hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	}

	storageConfiguration.Packfile.MaxSize = math.MaxUint64

	serializedConfig, err := storageConfiguration.ToBytes()
	if err != nil {
		return err
	}

	rd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serializedConfig))
	if err != nil {
		return err
	}
	wrappedConfig, err := io.ReadAll(rd)
	if err != nil {
		return err
	}

	location := klosetPath
	if !strings.HasPrefix(location, "ptar:") {
		location = "ptar://" + location
	}
	noSchemeLocation := strings.TrimPrefix(location, "ptar://")

	if _, err := os.Stat(noSchemeLocation); err == nil {
		if !overwrite {
			return fmt.Errorf("ptar archive %s already exists, use -overwrite to overwrite it", noSchemeLocation)
		} else {
			if err := os.Remove(noSchemeLocation); err != nil {
				return fmt.Errorf("could not remove existing ptar archive %s: %w", noSchemeLocation, err)
			}
		}
	}

	st, err := storage.Create(ctx.GetInner(), map[string]string{"location": location}, wrappedConfig)
	if err != nil {
		return err
	}

	repo, err = repository.New(ctx.GetInner(), key, st, wrappedConfig)
	if err != nil {
		return err
	}

	identifier := objects.RandomMAC()
	scanCache, err := repo.AppContext().GetCache().Scan(identifier)
	if err != nil {
		return err
	}

	defer scanCache.Close()

	repoWriter := repo.NewRepositoryWriter(scanCache, identifier, repository.PtarType, "")
	for i, syncTarget := range syncTargets {
		storeConfig, err := ctx.Config.GetRepository(syncTarget)
		if err != nil {
			return fmt.Errorf("source repository: %w", err)
		}

		peerStore, peerStoreSerializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
		if err != nil {
			return fmt.Errorf("could not open source store %s: %s", syncTarget, err)
		}

		srcCtx := appcontext.NewAppContextFrom(ctx)
		srcRepository, err := repository.New(srcCtx.GetInner(), syncSecrets[i], peerStore, peerStoreSerializedConfig)
		if err != nil {
			return fmt.Errorf("could not open source repository %s: %s", syncTarget, err)
		}

		if err := synchronize(ctx, srcRepository, repoWriter); err != nil {
			return err
		}
	}
	if err := backup(ctx, repoWriter, backupTargets); err != nil {
		return err
	}

	// We are done with everything we can now stop the backup routines.
	repoWriter.PackerManager.Wait()
	err = repoWriter.CommitTransaction(identifier)
	if err != nil {
		return err
	}

	if err := st.Close(ctx); err != nil {
		return err
	}

	return nil
}

func backup(ctx *appcontext.AppContext, repo *repository.RepositoryWriter, targets []string) error {
	for _, loc := range targets {
		opts := map[string]string{
			"location": loc,
		}
		if strings.HasPrefix(loc, "@") {
			remote, ok := ctx.Config.GetSource(loc[1:])
			if !ok {
				return fmt.Errorf("could not resolve importer: %s", loc)
			}
			if _, ok := remote["location"]; !ok {
				return fmt.Errorf("could not resolve importer location: %s", loc)
			}
			opts = remote
		}

		imp, err := importer.NewImporter(ctx.GetInner(), ctx.ImporterOpts(), opts)
		if err != nil {
			return err
		}

		backupOptions := &snapshot.BuilderOptions{
			NoCheckpoint: true,
			NoCommit:     true,
		}

		source, err := snapshot.NewSource(ctx, imp)
		if err != nil {
			return err
		}

		snap, err := snapshot.CreateWithRepositoryWriter(repo, backupOptions, objects.NilMac)
		if err != nil {
			return err
		}

		err = snap.Backup(source)
		if err != nil {
			return err
		}

		_, err = snap.PutSnapshot()
		if err != nil {
			return err
		}

		err = snap.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func synchronize(ctx *appcontext.AppContext, srcRepository *repository.Repository, dstRepository *repository.RepositoryWriter) error {
	srcLocateOptions := locate.NewDefaultLocateOptions()
	srcSnapshotIDs, err := locate.LocateSnapshotIDs(srcRepository, srcLocateOptions)
	if err != nil {
		return err
	}

	for _, snapshotID := range srcSnapshotIDs {
		if err := ctx.Err(); err != nil {
			return err
		}

		srcSnapshot, err := snapshot.Load(srcRepository, snapshotID)
		if err != nil {
			return err
		}
		defer srcSnapshot.Close()

		dstSnapshot, err := snapshot.CreateWithRepositoryWriter(dstRepository, &snapshot.BuilderOptions{
			NoCheckpoint: true,
			NoCommit:     true,
		}, srcSnapshot.Header.Identifier)
		if err != nil {
			return err
		}
		defer dstSnapshot.Close()

		// overwrite the header, we want to keep the original snapshot info
		dstSnapshot.Header = srcSnapshot.Header

		if err := srcSnapshot.Synchronize(dstSnapshot); err != nil {
			return err
		}
	}

	return nil
}
