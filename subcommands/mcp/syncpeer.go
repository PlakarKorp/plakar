/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
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

package mcp

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands/sync"
	"github.com/PlakarKorp/plakar/utils"
)

// syncSnapshotsInput mirrors "plakar sync to", restricted to peers named in the
// Plakar configuration: a model pushes to repositories the operator set up, it
// does not get to invent destinations. Pushing is the only direction offered,
// because pulling would let a client write snapshots into the repository it is
// serving.
type syncSnapshotsInput struct {
	Peer     string `json:"peer" jsonschema:"peer repository from the Plakar configuration, like @offsite"`
	Snapshot string `json:"snapshot,omitempty" jsonschema:"single snapshot ID to push, every snapshot by default"`
}

type syncSnapshotsOutput struct {
	Peer      string `json:"peer"`
	Direction string `json:"direction" jsonschema:"always to: this tool only pushes"`
	Synced    bool   `json:"synced"`
}

func syncSnapshots(ctx *appcontext.AppContext, repo *repository.Repository, in syncSnapshotsInput) (syncSnapshotsOutput, error) {
	if !strings.HasPrefix(in.Peer, "@") {
		return syncSnapshotsOutput{}, fmt.Errorf("peer must name a repository from the configuration, like @offsite")
	}

	peerSecret, err := peerRepositorySecret(ctx, in.Peer)
	if err != nil {
		return syncSnapshotsOutput{}, err
	}

	options := locate.NewDefaultLocateOptions()
	if in.Snapshot != "" {
		options.Filters.IDs = []string{in.Snapshot}
	}

	// Reuse the sync subcommand rather than reimplementing it; its Parse step is
	// bypassed because it may prompt for a passphrase, which has no place on a
	// stdio transport, so its fields are filled the way Parse would have.
	cmd := &sync.Sync{
		PeerRepositoryLocation: in.Peer,
		PeerRepositorySecret:   peerSecret,
		Direction:              "to",
		Cache:                  "vfs",
		SrcLocateOptions:       options,
	}
	cmd.RepositorySecret = ctx.GetSecret()

	status, err := cmd.Execute(ctx, repo)
	if err != nil {
		return syncSnapshotsOutput{}, err
	}
	if status != 0 {
		return syncSnapshotsOutput{}, fmt.Errorf("sync failed")
	}

	return syncSnapshotsOutput{Peer: in.Peer, Direction: "to", Synced: true}, nil
}

// peerRepositorySecret opens the peer's store configuration and derives its
// key from the configured passphrase. An encrypted peer with no configured
// passphrase is an error: prompting is not an option on a stdio transport.
func peerRepositorySecret(ctx *appcontext.AppContext, peer string) ([]byte, error) {
	if ctx.Config == nil {
		return nil, fmt.Errorf("no configuration loaded, cannot resolve %s", peer)
	}

	storeConfig, err := ctx.Config.GetRepository(peer)
	if err != nil {
		return nil, fmt.Errorf("peer store: %w", err)
	}

	passphrase, hasPassphrase := storeConfig["passphrase"]
	passphraseCmd, hasPassphraseCmd := storeConfig["passphrase_cmd"]
	delete(storeConfig, "passphrase")
	delete(storeConfig, "passphrase_cmd")

	peerStore, peerStoreSerializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return nil, fmt.Errorf("could not open peer store %s: %w", peer, err)
	}
	defer peerStore.Close(ctx)

	peerStoreConfig, err := storage.NewConfigurationFromWrappedBytes(peerStoreSerializedConfig)
	if err != nil {
		return nil, err
	}

	if err := utils.CheckPlaintext(storeConfig["location"], peerStoreConfig.Encryption != nil); err != nil {
		return nil, err
	}

	if peerStoreConfig.Encryption == nil {
		return nil, nil
	}

	if !hasPassphrase && hasPassphraseCmd {
		passphrase, err = utils.GetPassphraseFromCommand(passphraseCmd)
		if err != nil {
			return nil, fmt.Errorf("failed to read passphrase from command: %w", err)
		}
	} else if !hasPassphrase {
		return nil, fmt.Errorf("%s: peer repository is encrypted and has no passphrase configured", peer)
	}

	key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, []byte(passphrase))
	if err != nil {
		return nil, err
	}
	if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
		return nil, fmt.Errorf("%s: invalid passphrase", peer)
	}

	return key, nil
}
