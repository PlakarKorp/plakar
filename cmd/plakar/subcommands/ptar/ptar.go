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
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
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
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
)

func init() {
	subcommands.Register("ptar", parse_cmd_ptar)
}

func parse_cmd_ptar(ctx *appcontext.AppContext, args []string) (subcommands.Subcommand, error) {
	var opt_hashing string
	var opt_noencryption bool
	var opt_nocompression bool
	var opt_allowweak bool

	flags := flag.NewFlagSet("ptar", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: plakar [at /path/to/repository] %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       plakar [at @REPOSITORY] %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&opt_allowweak, "weak-passphrase", false, "allow weak passphrase to protect the repository")
	flags.StringVar(&opt_hashing, "hashing", hashing.DEFAULT_HASHING_ALGORITHM, "hashing algorithm to use for digests")
	flags.BoolVar(&opt_noencryption, "no-encryption", false, "disable transparent encryption")
	flags.BoolVar(&opt_nocompression, "no-compression", false, "disable transparent compression")
	flags.Parse(args)

	if flags.NArg() != 1 {
		return nil, fmt.Errorf("%s: too many parameters", flag.CommandLine.Name())
	}

	if hashing.GetHasher(strings.ToUpper(opt_hashing)) == nil {
		return nil, fmt.Errorf("%s: unknown hashing algorithm", flag.CommandLine.Name())
	}

	return &Ptar{
		AllowWeak:     opt_allowweak,
		Hashing:       opt_hashing,
		NoEncryption:  opt_noencryption,
		NoCompression: opt_nocompression,
		Location:      flags.Arg(0),
	}, nil
}

type Ptar struct {
	AllowWeak     bool
	Hashing       string
	NoEncryption  bool
	NoCompression bool
	Location      string
}

func (cmd *Ptar) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	storageConfiguration := storage.NewConfiguration()
	if cmd.NoCompression {
		storageConfiguration.Compression = nil
	} else {
		storageConfiguration.Compression = compression.NewDefaultConfiguration()
	}

	hashingConfiguration, err := hashing.LookupDefaultConfiguration(strings.ToUpper(cmd.Hashing))
	if err != nil {
		return 1, err
	}
	storageConfiguration.Hashing = *hashingConfiguration

	minEntropBits := 80.
	if cmd.AllowWeak {
		minEntropBits = 0.
	}

	var hasher hash.Hash
	if !cmd.NoEncryption {
		storageConfiguration.Encryption = encryption.NewDefaultConfiguration()

		var passphrase []byte

		envPassphrase := os.Getenv("PLAKAR_PASSPHRASE")
		if ctx.KeyFromFile == "" {
			if envPassphrase != "" {
				passphrase = []byte(envPassphrase)
			} else {
				for attempt := 0; attempt < 3; attempt++ {
					tmp, err := utils.GetPassphraseConfirm("repository", minEntropBits)
					if err != nil {
						fmt.Fprintf(os.Stderr, "%s\n", err)
						continue
					}
					passphrase = tmp
					break
				}
			}
		} else {
			passphrase = []byte(ctx.KeyFromFile)
		}

		if len(passphrase) == 0 {
			return 1, fmt.Errorf("can't encrypt the repository with an empty passphrase")
		}

		key, err := encryption.DeriveKey(storageConfiguration.Encryption.KDFParams, passphrase)
		if err != nil {
			return 1, err
		}

		canary, err := encryption.DeriveCanary(storageConfiguration.Encryption, key)
		if err != nil {
			return 1, err
		}
		storageConfiguration.Encryption.Canary = canary
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, key)
		ctx.SetSecret(key)
	} else {
		storageConfiguration.Encryption = nil
		hasher = hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	}

	storageConfiguration.Packfile.MaxSize = math.MaxUint64

	serializedConfig, err := storageConfiguration.ToBytes()
	if err != nil {
		return 1, err
	}

	rd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serializedConfig))
	if err != nil {
		return 1, err
	}
	wrappedConfig, err := io.ReadAll(rd)
	if err != nil {
		return 1, err
	}

	st, err := storage.Create(map[string]string{"location": repo.Location()}, wrappedConfig)
	if err != nil {
		panic(err)
	}

	repo, err = repository.New(ctx, st, wrappedConfig)
	if err != nil {
		panic(err)
	}

	imp, err := importer.NewImporter(map[string]string{"location": cmd.Location})
	if err != nil {
		panic(err)
	}

	snap, err := snapshot.Create(repo, repository.PtarType)
	if err != nil {
		panic(err)
	}

	backupOptions := &snapshot.BackupOptions{
		MaxConcurrency: 4,
		NoCheckpoint:   true,
	}

	err = snap.Backup(imp, backupOptions)
	if err != nil {
		panic(err)
	}

	if err := st.Close(); err != nil {
		return 1, err
	}

	return 0, nil
}
