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

package signify

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	gosignify "github.com/PlakarKorp/go-signify"
)

var (
	ErrNoTrustedKey = errors.New("no trusted key for this origin")
	ErrUntrusted    = errors.New("signature does not match any trusted key")
)

type TrustedKey struct {
	Name string // the trust-store basename, or a fixed name for builtin keys

	Key *gosignify.PublicKey

	Scopes []*Scope // a key with no scopes is authorised for nothing

	Builtin bool // compiled in, so not removable
}

func (k *TrustedKey) Authorises(origin string) bool {
	return slices.ContainsFunc(k.Scopes, func(s *Scope) bool {
		return s.Matches(origin)
	})
}

type TrustStore struct {
	keys []*TrustedKey
}

func NewTrustStore(keys ...*TrustedKey) *TrustStore {
	return &TrustStore{keys: slices.Clone(keys)}
}

func (t *TrustStore) Add(key *TrustedKey) {
	t.keys = append(t.keys, key)
}

func (t *TrustStore) Keys() []*TrustedKey {
	return slices.Clone(t.keys)
}

// For returns the keys authorised for origin. Empty means nothing may be
// installed from it.
func (t *TrustStore) For(origin string) []*TrustedKey {
	var out []*TrustedKey

	for _, k := range t.keys {
		if k.Authorises(origin) {
			out = append(out, k)
		}
	}

	return out
}

// Verify checks a signature against the keys authorised for origin. Selection
// is by key number: a signature names its own key.
func (t *TrustStore) Verify(origin string, msg []byte, sig *gosignify.Signature) (*TrustedKey, error) {
	candidates := t.For(origin)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoTrustedKey, origin)
	}

	for _, k := range candidates {
		if k.Key.KeyNum != sig.KeyNum {
			continue
		}

		if err := k.Key.Verify(msg, sig); err != nil {
			// Key numbers are unique, so no other key claims
			// this signature.
			return nil, fmt.Errorf("%w: %s", err, k.Name)
		}

		return k, nil
	}

	return nil, fmt.Errorf("%w: signed by unknown key %s", ErrUntrusted, sig.KeyNum)
}

// LoadTrustDir reads every *.pub in dir, with scopes from a <key>.scope
// sidecar. A key with no sidecar is authorised for nothing: scoping is not
// optional, and its absence is more likely a mistake than intent.
func LoadTrustDir(dir string) ([]*TrustedKey, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var keys []*TrustedKey

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}

		path := filepath.Join(dir, e.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		pk, err := gosignify.ParsePublicKey(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}

		scopes, err := loadScopes(strings.TrimSuffix(path, ".pub") + ".scope")
		if err != nil {
			return nil, err
		}

		keys = append(keys, &TrustedKey{
			Name:   strings.TrimSuffix(e.Name(), ".pub"),
			Key:    pk,
			Scopes: scopes,
		})
	}

	return keys, nil
}

// One scope per line; blank lines and "#" comments ignored.
func loadScopes(path string) ([]*Scope, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var scopes []*Scope

	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		s, err := ParseScope(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, i+1, err)
		}

		scopes = append(scopes, s)
	}

	return scopes, nil
}
