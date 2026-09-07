package signify

import (
	"strings"
	"testing"

	gosignify "github.com/PlakarKorp/go-signify"
)

// BuiltinKeys skips entries it cannot parse, so that a malformed compiled-in
// key cannot break plakar at runtime. That makes it silent, which is only safe
// if a test refuses to let a malformed key ship in the first place.
func TestBuiltinKeysAreWellFormed(t *testing.T) {
	if len(builtinKeys) == 0 {
		t.Skip("no builtin keys compiled in")
	}

	for _, b := range builtinKeys {
		t.Run(b.name, func(t *testing.T) {
			if b.name == "" {
				t.Error("builtin key has no name")
			}

			pk, err := gosignify.ParsePublicKey([]byte(b.key))
			if err != nil {
				t.Fatalf("does not parse: %v", err)
			}

			if len(pk.Key) != 32 {
				t.Errorf("key is %d bytes, want 32", len(pk.Key))
			}

			if len(b.scopes) == 0 {
				t.Error("no scopes: the key would be authorised for nothing")
			}

			for _, s := range b.scopes {
				if _, err := ParseScope(s); err != nil {
					t.Errorf("scope %q does not parse: %v", s, err)
				}
			}
		})
	}
}

// Every builtin key must survive the path BuiltinKeys takes, since anything it
// drops is silently untrusted.
func TestBuiltinKeysAllLoad(t *testing.T) {
	if got, want := len(BuiltinKeys()), len(builtinKeys); got != want {
		t.Fatalf("BuiltinKeys() returned %d keys, want %d — one was dropped", got, want)
	}

	for _, k := range BuiltinKeys() {
		if !k.Builtin {
			t.Errorf("key %q is not marked Builtin", k.Name)
		}

		if len(k.Scopes) == 0 {
			t.Errorf("key %q loaded with no scopes", k.Name)
		}
	}
}

// The publishing key must cover the whole dist tree, community and enterprise
// alike, and must not reach beyond plakar.io.
func TestPlakarKeyScope(t *testing.T) {
	var plakar *TrustedKey

	for _, k := range BuiltinKeys() {
		if strings.HasPrefix(k.Name, "plakar-") {
			plakar = k
			break
		}
	}

	if plakar == nil {
		t.Skip("no plakar builtin key")
	}

	authorised := []string{
		"https://plakar.io/dist/plugins/kloset/community/v1.1.0/s3/",
		"https://plakar.io/dist/plugins/kloset/enterprise/v1.1.0/s3/",
		LocalScope,
	}

	for _, origin := range authorised {
		if !plakar.Authorises(origin) {
			t.Errorf("key does not authorise %s", origin)
		}
	}

	refused := []string{
		"https://plakar.io/dist/releases/",
		"https://evil.example.com/dist/plugins/kloset/",
		"http://plakar.io/dist/plugins/kloset/",
		"https://plakar.io/dist/plugins/kloset-evil/",
	}

	for _, origin := range refused {
		if plakar.Authorises(origin) {
			t.Errorf("key wrongly authorises %s", origin)
		}
	}
}

// The comment is what a user sees when inspecting trust, so it should identify
// the key rather than leak where it was kept.
func TestBuiltinKeyComments(t *testing.T) {
	for _, k := range BuiltinKeys() {
		if strings.Contains(k.Key.Comment, "/") {
			t.Errorf("key %q comment contains a path: %q", k.Name, k.Key.Comment)
		}
	}
}

// Builtin key names must be unique and dated, so that a rotation adds a key
// rather than shadowing one: two entries called "plakar" would be
// indistinguishable in `pkg list` and in error messages.
func TestBuiltinKeyNamesAreDistinctAndDated(t *testing.T) {
	seen := map[string]bool{}

	for _, k := range BuiltinKeys() {
		if seen[k.Name] {
			t.Errorf("duplicate builtin key name %q", k.Name)
		}
		seen[k.Name] = true

		if strings.HasPrefix(k.Name, "plakar") && !strings.Contains(k.Name, "-") {
			t.Errorf("key %q is undated: rotation needs plakar-YYYYMMDD", k.Name)
		}
	}
}

// The property rotation depends on: several trusted keys coexist, and a
// signature is routed to its own key by key number without anyone naming it.
func TestRotationSelectsByKeyNum(t *testing.T) {
	outgoing, err := gosignify.ParsePublicKey(fixture(t, "plakar-20260815.pub"))
	if err != nil {
		t.Fatal(err)
	}

	incoming, err := gosignify.ParsePublicKey(fixture(t, "other.pub"))
	if err != nil {
		t.Fatal(err)
	}

	if outgoing.KeyNum == incoming.KeyNum {
		t.Fatal("fixtures share a key number; the test proves nothing")
	}

	scope := mustScope(t, testOrigin)

	store := NewTrustStore(
		&TrustedKey{Name: "plakar-20260815", Key: outgoing, Scopes: []*Scope{scope}},
		&TrustedKey{Name: "plakar-20270101", Key: incoming, Scopes: []*Scope{scope}},
	)

	// A signature made by the outgoing key still verifies while both are
	// trusted — which is what a deprecation window is for.
	sig, err := gosignify.ParseSignature(fixture(t, "art.sum.sig"))
	if err != nil {
		t.Fatal(err)
	}

	key, err := store.Verify(testOrigin, sig.Message, sig)
	if err != nil {
		t.Fatalf("outgoing key failed to verify during rotation: %v", err)
	}

	if key.Name != "plakar-20260815" {
		t.Errorf("verified by %q, want the outgoing key", key.Name)
	}
}
