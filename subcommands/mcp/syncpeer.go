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

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands/sync"
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

	// Opening the peer validates it and derives its secret; prompting is
	// disabled because stdio carries the protocol stream, so an encrypted peer
	// must have its passphrase configured.
	peerStore, _, peerSecret, err := sync.OpenPeer(ctx, in.Peer, false)
	if err != nil {
		return syncSnapshotsOutput{}, err
	}
	peerStore.Close(ctx)

	options := locate.NewDefaultLocateOptions()
	if in.Snapshot != "" {
		options.Filters.IDs = []string{in.Snapshot}
	}

	// Reuse the sync subcommand rather than reimplementing it; its Parse step
	// is skipped, so its fields are filled the way Parse would have.
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
		return syncSnapshotsOutput{}, fmt.Errorf("sync to %s: %w", in.Peer, err)
	}
	if status != 0 {
		return syncSnapshotsOutput{}, fmt.Errorf("sync to %s failed", in.Peer)
	}

	return syncSnapshotsOutput{Peer: in.Peer, Direction: "to", Synced: true}, nil
}
