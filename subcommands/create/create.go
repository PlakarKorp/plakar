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

package create

import (
	"bytes"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"github.com/PlakarKorp/kloset/compression"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Create{} }, subcommands.BeforeRepositoryWithStorage, "create")
}

func (cmd *Create) Parse(ctx *appcontext.AppContext, args []string) error {
	var allow_weak bool

	flags := flag.NewFlagSet("create", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: plakar [at /path/to/repository] %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       plakar [at @REPOSITORY] %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&allow_weak, "weak-passphrase", false, "allow weak passphrase to protect the repository")
	flags.StringVar(&cmd.Hashing, "hashing", hashing.DEFAULT_HASHING_ALGORITHM, "hashing algorithm to use for digests")
	flags.BoolVar(&cmd.NoEncryption, "plaintext", false, "disable transparent encryption")
	flags.BoolVar(&cmd.NoCompression, "no-compression", false, "disable transparent compression")
	flags.Parse(args)

	if flags.NArg() != 0 {
		return fmt.Errorf("%s: too many parameters", flag.CommandLine.Name())
	}

	if hashing.GetHasher(strings.ToUpper(cmd.Hashing)) == nil {
		return fmt.Errorf("%s: unknown hashing algorithm", flag.CommandLine.Name())
	}

	minEntropBits := 80.
	if allow_weak {
		minEntropBits = 0.
	}

	if !cmd.NoEncryption {
		var passphrase []byte

		if ctx.KeyFromFile == "" {
			for range 3 {
				tmp, err := utils.GetPassphraseConfirm("repository", minEntropBits)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", err)
					continue
				}
				passphrase = tmp
				break
			}
		} else {
			passphrase = []byte(ctx.KeyFromFile)
		}

		if len(passphrase) == 0 {
			return fmt.Errorf("can't encrypt the repository with an empty passphrase")
		}

		cmd.RepositorySecret = passphrase
	}

	return nil
}

type Create struct {
	subcommands.SubcommandBase

	Hashing       string
	NoEncryption  bool
	NoCompression bool
}

func (cmd *Create) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
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

	var hasher hash.Hash
	if !cmd.NoEncryption {
		key, err := encryption.DeriveKey(storageConfiguration.Encryption.KDFParams,
			cmd.RepositorySecret)
		if err != nil {
			return 1, err
		}

		canary, err := encryption.DeriveCanary(storageConfiguration.Encryption, key)
		if err != nil {
			return 1, err
		}
		storageConfiguration.Encryption.Canary = canary
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, key)
	} else {
		storageConfiguration.Encryption = nil
		hasher = hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	}

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

	if err := repo.Store().Create(ctx, wrappedConfig); err != nil {
		return 1, err
	}

	return 0, nil
}
