/*
 * Copyright (c) 2026 Gilles Chehade <gilles@poolp.org>
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

package sync

import (
	"fmt"
	"os"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
)

// OpenPeer opens the peer repository at location and derives its secret,
// asking for a passphrase only when prompt is set.
func OpenPeer(ctx *appcontext.AppContext, location string, prompt bool) (_ storage.Store, _ []byte, _ []byte, err error) {
	storeConfig, err := ctx.Config.GetRepository(location)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("peer store: %w", err)
	}

	pass, hasPass := storeConfig["passphrase"]
	delete(storeConfig, "passphrase")
	passCmd, hasPassCmd := storeConfig["passphrase_cmd"]
	delete(storeConfig, "passphrase_cmd")

	peerStore, peerStoreSerializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return nil, nil, nil, err
	}
	defer func() {
		if err != nil {
			peerStore.Close(ctx)
		}
	}()

	peerStoreConfig, err := storage.NewConfigurationFromWrappedBytes(peerStoreSerializedConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	if err := utils.CheckPlaintext(storeConfig["location"], peerStoreConfig.Encryption != nil); err != nil {
		return nil, nil, nil, err
	}

	var peerSecret []byte
	if peerStoreConfig.Encryption != nil {
		if hasPass {
			key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, []byte(pass))
			if err != nil {
				return nil, nil, nil, err
			}
			if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
				return nil, nil, nil, fmt.Errorf("invalid passphrase")
			}
			peerSecret = key
		} else if hasPassCmd {
			passphrase, err := utils.GetPassphraseFromCommand(passCmd)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to read passphrase from command: %w", err)
			}
			key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, []byte(passphrase))
			if err != nil {
				return nil, nil, nil, err
			}
			if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
				return nil, nil, nil, fmt.Errorf("invalid passphrase")
			}
			peerSecret = key
		} else if prompt {
			for {
				passphrase, err := utils.GetPassphrase("destination store")
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", err)
					continue
				}

				key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, passphrase)
				if err != nil {
					return nil, nil, nil, err
				}
				if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
					return nil, nil, nil, fmt.Errorf("invalid passphrase")
				}
				peerSecret = key
				break
			}
		} else {
			return nil, nil, nil, fmt.Errorf("%s: no passphrase configured", location)
		}
	}

	return peerStore, peerStoreSerializedConfig, peerSecret, nil
}
