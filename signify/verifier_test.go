package signify

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	gosignify "github.com/PlakarKorp/go-signify"
)

const testOrigin = "https://plakar.io/dist/plugins/kloset/"

// plakarStore builds a trust store holding the real fixture key, scoped to the
// test origin and to local installs.
func plakarStore(t *testing.T, scopes ...string) *TrustStore {
	t.Helper()

	if len(scopes) == 0 {
		scopes = []string{testOrigin, LocalScope}
	}

	pk, err := gosignify.ParsePublicKey(fixture(t, "plakar-20260815.pub"))
	if err != nil {
		t.Fatal(err)
	}

	parsed := make([]*Scope, 0, len(scopes))
	for _, s := range scopes {
		parsed = append(parsed, mustScope(t, s))
	}

	return NewTrustStore(&TrustedKey{
		Name:    "plakar-20260815",
		Key:     pk,
		Scopes:  parsed,
		Builtin: true,
	})
}

const artifactName = "s3_v1.1.2_linux_amd64.ptar"

// The happy path, end to end, against artifacts signed by the real toolchain.
func TestVerifyArtifact(t *testing.T) {
	v := NewVerifier(plakarStore(t))

	res, err := v.VerifyArtifact(
		testOrigin,
		artifactName,
		fixture(t, "art.sum.sig"),
		bytes.NewReader(fixture(t, artifactName)),
	)
	if err != nil {
		t.Fatalf("VerifyArtifact on a genuine artifact: %v", err)
	}

	if res.Key.Name != "plakar-20260815" {
		t.Errorf("verified by %q, want plakar-20260815", res.Key.Name)
	}

	if res.Filename != artifactName {
		t.Errorf("filename = %q, want %q", res.Filename, artifactName)
	}
}

// Tampering with the artifact must be caught by the digest comparison, even
// though the signature itself is untouched and still valid.
func TestVerifyRejectsTamperedArtifact(t *testing.T) {
	v := NewVerifier(plakarStore(t))

	_, err := v.VerifyArtifact(
		testOrigin,
		artifactName,
		fixture(t, "art.sum.sig"),
		strings.NewReader("this is NOT the artifact content\n"),
	)

	if !errors.Is(err, gosignify.ErrDigestMismatch) {
		t.Errorf("got %v, want ErrDigestMismatch", err)
	}
}

// The attack the filename binding exists to stop: a genuine, validly signed
// artifact served under a different package's name.
func TestVerifyRejectsSubstitutedPackage(t *testing.T) {
	v := NewVerifier(plakarStore(t))

	_, err := v.VerifyArtifact(
		testOrigin,
		"postgresql_v1.0.0_linux_amd64.ptar", // asked for postgresql...
		fixture(t, "art.sum.sig"),            // ...served s3's signature
		bytes.NewReader(fixture(t, artifactName)),
	)

	if !errors.Is(err, ErrNameMismatch) {
		t.Errorf("got %v, want ErrNameMismatch", err)
	}
}

// A key trusted for one registry must not vouch for another.
func TestVerifyRejectsOutOfScopeOrigin(t *testing.T) {
	v := NewVerifier(plakarStore(t, testOrigin))

	_, err := v.VerifyArtifact(
		"https://evil.example.com/plugins/",
		artifactName,
		fixture(t, "art.sum.sig"),
		bytes.NewReader(fixture(t, artifactName)),
	)

	if !errors.Is(err, ErrNoTrustedKey) {
		t.Errorf("got %v, want ErrNoTrustedKey", err)
	}
}

// A key scoped to a registry must not validate a file handed over locally.
func TestVerifyRejectsLocalWhenNotScopedForIt(t *testing.T) {
	v := NewVerifier(plakarStore(t, testOrigin))

	_, err := v.VerifyArtifact(
		LocalScope,
		artifactName,
		fixture(t, "art.sum.sig"),
		bytes.NewReader(fixture(t, artifactName)),
	)

	if !errors.Is(err, ErrNoTrustedKey) {
		t.Errorf("got %v, want ErrNoTrustedKey", err)
	}
}

func TestVerifyAcceptsLocalWhenScopedForIt(t *testing.T) {
	v := NewVerifier(plakarStore(t, LocalScope))

	if _, err := v.VerifyArtifact(
		LocalScope,
		artifactName,
		fixture(t, "art.sum.sig"),
		bytes.NewReader(fixture(t, artifactName)),
	); err != nil {
		t.Errorf("VerifyArtifact: %v", err)
	}
}

// An artifact signed by a key we do not trust at all.
func TestVerifyRejectsUnknownKey(t *testing.T) {
	other, err := gosignify.ParsePublicKey(fixture(t, "other.pub"))
	if err != nil {
		t.Fatal(err)
	}

	store := NewTrustStore(&TrustedKey{
		Name:   "other",
		Key:    other,
		Scopes: []*Scope{mustScope(t, testOrigin)},
	})

	_, err = NewVerifier(store).VerifyArtifact(
		testOrigin,
		artifactName,
		fixture(t, "art.sum.sig"),
		bytes.NewReader(fixture(t, artifactName)),
	)

	if !errors.Is(err, ErrUntrusted) {
		t.Errorf("got %v, want ErrUntrusted", err)
	}
}

func TestVerifyRejectsUnsigned(t *testing.T) {
	v := NewVerifier(plakarStore(t))

	_, err := v.VerifyArtifact(testOrigin, artifactName, nil,
		bytes.NewReader(fixture(t, artifactName)))

	if !errors.Is(err, ErrUnsigned) {
		t.Errorf("got %v, want ErrUnsigned", err)
	}
}

func TestAllowUnsignedIsOptIn(t *testing.T) {
	v := NewVerifier(plakarStore(t))
	v.SetAllowUnsigned(true)

	res, err := v.VerifyArtifact(testOrigin, artifactName, nil,
		bytes.NewReader(fixture(t, artifactName)))
	if err != nil {
		t.Fatalf("with AllowUnsigned: %v", err)
	}

	if res != nil {
		t.Errorf("expected no verification result for an unsigned artifact, got %+v", res)
	}
}

// -allow-unsigned must not become a blanket bypass: a package that ships a
// signature is still verified, and a bad one is still fatal.
func TestAllowUnsignedStillVerifiesSignedArtifacts(t *testing.T) {
	v := NewVerifier(plakarStore(t))
	v.SetAllowUnsigned(true)

	// Genuine artifact under the wrong name: still rejected.
	_, err := v.VerifyArtifact(testOrigin, "postgresql_v1.0.0_linux_amd64.ptar",
		fixture(t, "art.sum.sig"), bytes.NewReader(fixture(t, artifactName)))
	if !errors.Is(err, ErrNameMismatch) {
		t.Errorf("got %v, want ErrNameMismatch", err)
	}

	// Tampered content with a valid signature: still rejected.
	_, err = v.VerifyArtifact(testOrigin, artifactName,
		fixture(t, "art.sum.sig"), strings.NewReader("tampered"))
	if !errors.Is(err, gosignify.ErrDigestMismatch) {
		t.Errorf("got %v, want ErrDigestMismatch", err)
	}

	// Signed by a key we do not trust: still rejected.
	other, err := gosignify.ParsePublicKey(fixture(t, "other.pub"))
	if err != nil {
		t.Fatal(err)
	}

	store := NewTrustStore(&TrustedKey{
		Name: "other", Key: other, Scopes: []*Scope{mustScope(t, testOrigin)},
	})

	v2 := NewVerifier(store)
	v2.SetAllowUnsigned(true)

	if _, err := v2.VerifyArtifact(testOrigin, artifactName,
		fixture(t, "art.sum.sig"), bytes.NewReader(fixture(t, artifactName))); !errors.Is(err, ErrUntrusted) {
		t.Errorf("got %v, want ErrUntrusted", err)
	}
}

// A detached signature carries no checksum, so there is nothing to bind the
// artifact to and it must be refused rather than accepted on the strength of
// the signature alone.
func TestVerifyRejectsDetachedSignature(t *testing.T) {
	lines := strings.SplitN(string(fixture(t, "art.sum.sig")), "\n", 3)
	detached := []byte(lines[0] + "\n" + lines[1] + "\n")

	_, err := NewVerifier(plakarStore(t)).VerifyArtifact(
		testOrigin, artifactName, detached,
		bytes.NewReader(fixture(t, artifactName)),
	)

	if !errors.Is(err, ErrNoChecksum) {
		t.Errorf("got %v, want ErrNoChecksum", err)
	}
}

func TestVerifyRejectsNonSHA256Checksum(t *testing.T) {
	_, sk, err := gosignify.GenerateKey("test")
	if err != nil {
		t.Fatal(err)
	}

	sums := "MD5 (" + artifactName + ") = " + strings.Repeat("a", 32) + "\n"

	sig, err := sk.SignEmbedded([]byte(sums), "c").Marshal()
	if err != nil {
		t.Fatal(err)
	}

	store := NewTrustStore(&TrustedKey{
		Name: "t", Key: sk.Public(), Scopes: []*Scope{mustScope(t, testOrigin)},
	})

	_, err = NewVerifier(store).VerifyArtifact(testOrigin, artifactName, sig,
		bytes.NewReader(fixture(t, artifactName)))

	if !errors.Is(err, ErrBadChecksum) {
		t.Errorf("got %v, want ErrBadChecksum", err)
	}
}

func TestTrustStoreFor(t *testing.T) {
	store := plakarStore(t, testOrigin)

	if got := store.For(testOrigin + "community/v1.1.0/s3/"); len(got) != 1 {
		t.Errorf("expected the key to cover a path below its scope, got %d keys", len(got))
	}

	if got := store.For("https://elsewhere.example/"); len(got) != 0 {
		t.Errorf("expected no keys for an unrelated origin, got %d", len(got))
	}
}
