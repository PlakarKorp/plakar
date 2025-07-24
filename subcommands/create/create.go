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

type Profile struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	PackfileMaxSize    uint64 `json:"packfile_max_size"`
	ChunkingMinSize    uint32 `json:"chunking_min_size"`
	ChunkingNormalSize uint32 `json:"chunking_normal_size"`
	ChunkingMaxSize    uint32 `json:"chunking_max_size"`
}

var profiles = []Profile{
	{
		Name:               "standard",
		Description:        "General-purpose profile for configs, documents, and most typical workloads",
		PackfileMaxSize:    64 << 20,
		ChunkingMinSize:    4 << 10,
		ChunkingNormalSize: 16 << 10,
		ChunkingMaxSize:    64 << 10,
	},
	{
		Name:               "small",
		Description:        "Maximizes deduplication; best for small, redundant files or systems with heavy reuse",
		PackfileMaxSize:    32 << 20,
		ChunkingMinSize:    2 << 10,
		ChunkingNormalSize: 8 << 10,
		ChunkingMaxSize:    32 << 10,
	},
	{
		Name:               "fast",
		Description:        "Speeds up snapshot/restore for mid-size and large files; less granular dedup",
		PackfileMaxSize:    64 << 20,
		ChunkingMinSize:    8 << 10,
		ChunkingNormalSize: 32 << 10,
		ChunkingMaxSize:    128 << 10,
	},
	{
		Name:               "archive",
		Description:        "For large, infrequently changing files; reduces indexing and metadata overhead",
		PackfileMaxSize:    128 << 20,
		ChunkingMinSize:    16 << 10,
		ChunkingNormalSize: 64 << 10,
		ChunkingMaxSize:    256 << 10,
	},
	{
		Name:               "blob",
		Description:        "Optimized for huge immutable blobs (e.g. video, .tar.gz); fastest, lowest overhead",
		PackfileMaxSize:    512 << 20,
		ChunkingMinSize:    512 << 10,
		ChunkingNormalSize: 2 << 20,
		ChunkingMaxSize:    8 << 20,
	},
}

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
	flags.StringVar(&cmd.Profile, "profile", "standard", "repository profile to use (standard, small, fast, archive, blob)")
	flags.Parse(args)

	if flags.NArg() != 0 {
		return fmt.Errorf("%s: too many parameters", flag.CommandLine.Name())
	}

	found := false
	for _, profile := range profiles {
		if strings.EqualFold(cmd.Profile, profile.Name) {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%s: unknown profile '%s'", flag.CommandLine.Name(), cmd.Profile)
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

	Profile string
}

func (cmd *Create) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var profile *Profile
	for _, item := range profiles {
		if strings.EqualFold(cmd.Profile, item.Name) {
			profile = &item
			break
		}
	}

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

	storageConfiguration.Packfile.MaxSize = profile.PackfileMaxSize
	storageConfiguration.Chunking.MinSize = profile.ChunkingMinSize
	storageConfiguration.Chunking.NormalSize = profile.ChunkingNormalSize
	storageConfiguration.Chunking.MaxSize = profile.ChunkingMaxSize

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
