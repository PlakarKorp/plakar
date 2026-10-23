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

package fuzz

import (
	"bytes"
	"strings"
	"testing"

	gosignify "github.com/PlakarKorp/go-signify"

	"github.com/PlakarKorp/plakar/signify"
)

// A trust scope as written in a <key>.scope sidecar, matched against the
// origin of a package being verified.
func FuzzSignifyScope(f *testing.F) {
	f.Add("https://plugins.plakar.io/", "https://plugins.plakar.io/kloset/x.ptar")
	f.Add("local", "local")
	f.Add("HTTPS://Example.test/a/../b/", "https://example.test/b/c")

	f.Fuzz(func(t *testing.T, scope, origin string) {
		s, err := signify.ParseScope(scope)
		if err != nil {
			return
		}
		_ = s.Matches(origin)
		_ = s.String()
		key := &signify.TrustedKey{Scopes: []*signify.Scope{s}}
		_ = key.Authorises(origin)
	})
}

// A plugin package and its signify signature as `plakar pkg` fetches them
// from a registry, verified against a trust store that holds one key.
// Everything from the signature file inwards is attacker-controlled: the
// envelope, the signed checksum list, and the content hashed against it.
func FuzzSignifyArtifact(f *testing.F) {
	pk, sk := testKey(f)
	scope, err := signify.ParseScope("https://example.test/")
	if err != nil {
		f.Fatal(err)
	}
	store := signify.NewTrustStore(&signify.TrustedKey{
		Name:   "fuzz",
		Key:    pk,
		Scopes: []*signify.Scope{scope},
	})
	verifier := signify.NewVerifier(store)

	content := []byte("a plugin package")
	sum, err := gosignify.ChecksumFile("plugin.ptar", bytes.NewReader(content))
	if err != nil {
		f.Fatal(err)
	}
	sig, err := sk.SignEmbedded([]byte(sum.String()+"\n"), "fuzz").Marshal()
	if err != nil {
		f.Fatal(err)
	}
	f.Add(sig, content, "plugin.ptar")
	f.Add(sig, []byte("tampered"), "plugin.ptar")
	f.Add(sig, content, "other.ptar")
	f.Add([]byte("untrusted comment: x\n"), content, "plugin.ptar")

	f.Fuzz(func(t *testing.T, sig, content []byte, filename string) {
		origin := "https://example.test/" + filename
		_, _ = verifier.VerifyArtifact(origin, filename, sig, bytes.NewReader(content))
		_, _ = gosignify.ParsePublicKey(sig)
		_, _ = gosignify.ParseChecksums(sig)
	})
}

// testKey returns the signify keypair the artifact target trusts, derived
// from a fixed seed so runs are reproducible. It signs nothing outside this
// file.
func testKey(f *testing.F) (*gosignify.PublicKey, *gosignify.SecretKey) {
	entropy := strings.NewReader(strings.Repeat("plakar fuzz key ", 8))
	pk, sk, err := gosignify.GenerateKeyFrom(entropy, "plakar fuzz key")
	if err != nil {
		f.Fatalf("deterministic key generation: %v", err)
	}
	return pk, sk
}
