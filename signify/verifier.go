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
	"io"
	"path/filepath"
	"strings"
	"sync/atomic"

	gosignify "github.com/PlakarKorp/go-signify"
	"github.com/PlakarKorp/pkg"
)

var (
	ErrUnsigned       = errors.New("package is not signed")
	ErrNameMismatch   = errors.New("signature covers a different file")
	ErrDigestMismatch = errors.New("package contents do not match the signed checksum")
	ErrNoChecksum     = errors.New("signature carries no checksum")
	ErrBadChecksum    = errors.New("unusable signed checksum")
)

type Verifier struct {
	store *TrustStore

	allowUnsigned atomic.Bool
}

func NewVerifier(store *TrustStore) *Verifier {
	return &Verifier{store: store}
}

// Opt-in per run only. A signed artifact is still verified, and a bad
// signature is still fatal.
func (v *Verifier) SetAllowUnsigned(allow bool) {
	v.allowUnsigned.Store(allow)
}

type Result struct {
	Key      *TrustedKey
	Filename string
	Digest   string
}

// VerifyArtifact checks that content is the file named by filename, signed by
// a key trusted for origin.
//
// The order is deliberate: verify the signature, then compare the signed
// filename against the one requested, then hash. The filename comparison is
// what binds a signature to a package — without it, substituting an artifact
// and its signature together passes every other check. The standalone .sum is
// never consulted: it is unsigned, and the signature carries the copy that
// matters.
func (v *Verifier) VerifyArtifact(origin, filename string, sig []byte, content io.Reader) (*Result, error) {
	if len(sig) == 0 {
		if v.allowUnsigned.Load() {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %s", ErrUnsigned, filename)
	}

	parsed, err := gosignify.ParseSignature(sig)
	if err != nil {
		return nil, err
	}

	if parsed.Message == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoChecksum, filename)
	}

	key, err := v.store.Verify(origin, parsed.Message, parsed)
	if err != nil {
		return nil, err
	}

	sums, err := gosignify.ParseChecksums(parsed.Message)
	if err != nil {
		return nil, err
	}

	want, err := sums.Find(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNameMismatch, err)
	}

	// The library accepts any algorithm a checksum file names; a package
	// signature must be SHA256. Rejected here rather than left to the
	// digest comparison, which would report a mismatch instead of the
	// actual problem.
	if !strings.EqualFold(want.Algorithm, "SHA256") {
		return nil, fmt.Errorf("%w: %s is signed with %s, want SHA256",
			ErrBadChecksum, filename, want.Algorithm)
	}

	if err := want.Verify(content); err != nil {
		return nil, err
	}

	return &Result{Key: key, Filename: filename, Digest: want.Digest}, nil
}

// Verify implements pkg.Verifier. Errors wrap pkg.ErrUnverified so callers can
// tell a rejection from a transport failure.
func (v *Verifier) Verify(artifact *pkg.Artifact, rd io.Reader) error {
	if _, err := v.VerifyArtifact(artifact.Origin, artifact.Filename, artifact.Signature, rd); err != nil {
		return fmt.Errorf("%w: %s: %w", pkg.ErrUnverified, artifact.Filename, err)
	}

	return nil
}

// Signer names the key that produced sig, by key number, so a key that has
// since been rotated out is still named. This says who signed, not that the
// signature is good.
func (v *Verifier) Signer(sig []byte) string {
	if len(sig) == 0 {
		return ""
	}

	parsed, err := gosignify.ParseSignature(sig)
	if err != nil {
		return "unreadable signature"
	}

	for _, k := range v.store.Keys() {
		if k.Key.KeyNum == parsed.KeyNum {
			return k.Name
		}
	}

	return "unknown key " + parsed.KeyNum.String()
}

// BuiltinKeys returns the keys compiled into this binary. Embedded rather than
// read from disk: a key fetched from where a package could also reach would
// verify nothing.
func BuiltinKeys() []*TrustedKey {
	keys := make([]*TrustedKey, 0, len(builtinKeys))

	for _, b := range builtinKeys {
		pk, err := gosignify.ParsePublicKey([]byte(b.key))
		if err != nil {
			// A build-time mistake, not a runtime condition; the
			// tests check every builtin key parses.
			continue
		}

		scopes := make([]*Scope, 0, len(b.scopes))
		for _, s := range b.scopes {
			sc, err := ParseScope(s)
			if err != nil {
				continue
			}
			scopes = append(scopes, sc)
		}

		keys = append(keys, &TrustedKey{
			Name:    b.name,
			Key:     pk,
			Scopes:  scopes,
			Builtin: true,
		})
	}

	return keys
}

// LoadTrustStore builds this run's trust store: the compiled-in keys plus any
// the user added under configDir.
func LoadTrustStore(configDir string) (*TrustStore, error) {
	store := NewTrustStore(BuiltinKeys()...)

	userKeys, err := LoadTrustDir(filepath.Join(configDir, "trust"))
	if err != nil {
		return nil, err
	}

	for _, k := range userKeys {
		store.Add(k)
	}

	return store, nil
}
