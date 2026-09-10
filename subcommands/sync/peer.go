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

package sync

import (
	"fmt"
	"os"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
)

// OpenPeer resolves, opens and unlocks the peer repository at location, which
// may be a path, an URI or a @name from the configuration. It returns the open
// store, its serialized configuration and the derived secret, nil when the
// peer is not encrypted. With prompt, a missing passphrase is asked for
// interactively; without it, an encrypted peer whose passphrase is not
// configured is an error, for callers whose standard streams carry something
// else than a terminal.
func OpenPeer(ctx *appcontext.AppContext, location string, prompt bool) (storage.Store, []byte, []byte, error) {
	if ctx.Config == nil {
		return nil, nil, nil, fmt.Errorf("no configuration loaded, cannot resolve %s", location)
	}

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

	peerStoreConfig, err := storage.NewConfigurationFromWrappedBytes(peerStoreSerializedConfig)
	if err != nil {
		peerStore.Close(ctx)
		return nil, nil, nil, err
	}

	if err := utils.CheckPlaintext(storeConfig["location"], peerStoreConfig.Encryption != nil); err != nil {
		peerStore.Close(ctx)
		return nil, nil, nil, err
	}

	if peerStoreConfig.Encryption == nil {
		return peerStore, peerStoreSerializedConfig, nil, nil
	}

	derive := func(passphrase []byte) ([]byte, error) {
		key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, passphrase)
		if err != nil {
			return nil, err
		}
		if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
			return nil, fmt.Errorf("invalid passphrase")
		}
		return key, nil
	}

	var peerSecret []byte
	switch {
	case hasPass:
		peerSecret, err = derive([]byte(pass))
	case hasPassCmd:
		var passphrase string
		if passphrase, err = utils.GetPassphraseFromCommand(passCmd); err != nil {
			err = fmt.Errorf("failed to read passphrase from command: %w", err)
			break
		}
		peerSecret, err = derive([]byte(passphrase))
	case prompt:
		for {
			passphrase, promptErr := utils.GetPassphrase("destination store")
			if promptErr != nil {
				fmt.Fprintf(os.Stderr, "%s\n", promptErr)
				continue
			}
			peerSecret, err = derive(passphrase)
			break
		}
	default:
		err = fmt.Errorf("%s: peer repository is encrypted and has no passphrase configured", location)
	}
	if err != nil {
		peerStore.Close(ctx)
		return nil, nil, nil, err
	}

	return peerStore, peerStoreSerializedConfig, peerSecret, nil
}
