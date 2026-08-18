package signify

import (
	"errors"
	"testing"
)

func mustScope(t *testing.T, s string) *Scope {
	t.Helper()

	sc, err := ParseScope(s)
	if err != nil {
		t.Fatalf("ParseScope(%q): %v", s, err)
	}

	return sc
}

func TestScopeMatching(t *testing.T) {
	const base = "https://plakar.io/dist/plugins/kloset/"

	cases := []struct {
		name   string
		scope  string
		origin string
		want   bool
	}{
		{"exact", base, base, true},
		{"below the scope", base, base + "community/v1.1.0/s3/", true},
		{"scope root matches all", "https://plakar.io/", base, true},

		// The case a plain string prefix would get wrong.
		{"sibling with a shared prefix", base, "https://plakar.io/dist/plugins/kloset-evil/", false},
		{"prefix of a longer segment", "https://plakar.io/dist/plug/", "https://plakar.io/dist/plugins/", false},

		{"different host", base, "https://evil.io/dist/plugins/kloset/", false},
		{"subdomain is not covered", "https://plakar.io/", "https://evil.plakar.io/", false},
		{"host is a suffix", "https://plakar.io/", "https://notplakar.io/", false},
		{"scheme downgrade", base, "http://plakar.io/dist/plugins/kloset/", false},
		{"above the scope", base + "community/", base, false},

		{"host case is ignored", "https://PLAKAR.IO/dist/", "https://plakar.io/dist/", true},
		{"default port elided", "https://plakar.io:443/dist/", "https://plakar.io/dist/", true},
		{"non-default port must match", "https://plakar.io:8443/dist/", "https://plakar.io/dist/", false},

		{"credentials in origin rejected", "https://plakar.io/dist/", "https://u:p@plakar.io/dist/", false},
		{"local never matches a URL", LocalScope, base, false},
		{"URL never matches local", base, LocalScope, false},
		{"local matches local", LocalScope, LocalScope, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mustScope(t, tc.scope).Matches(tc.origin)
			if got != tc.want {
				t.Errorf("scope %q matching origin %q = %v, want %v",
					tc.scope, tc.origin, got, tc.want)
			}
		})
	}
}

// Path traversal in an origin must not be able to escape a scope.
func TestScopeRejectsTraversal(t *testing.T) {
	scope := mustScope(t, "https://plakar.io/dist/plugins/kloset/")

	for _, origin := range []string{
		"https://plakar.io/dist/plugins/kloset/../../../etc/",
		"https://plakar.io/dist/plugins/kloset/..%2f..%2fevil/",
	} {
		if scope.Matches(origin) {
			t.Errorf("origin %q escaped the scope", origin)
		}
	}
}

func TestParseScopeRejectsUnsafeInput(t *testing.T) {
	for _, s := range []string{
		"ftp://plakar.io/dist/",
		"file:///etc/",
		"https://",
		"https://u:p@plakar.io/dist/",
		"https://plakar.io/dist/?x=1",
		"https://plakar.io/dist/#frag",
		"plakar.io/dist/",
	} {
		if _, err := ParseScope(s); !errors.Is(err, ErrInvalidScope) {
			t.Errorf("ParseScope(%q) = %v, want ErrInvalidScope", s, err)
		}
	}
}

func TestScopeRoundTrip(t *testing.T) {
	for _, s := range []string{
		LocalScope,
		"https://plakar.io/dist/plugins/kloset/",
		"http://registry.internal:8080/plugins/",
	} {
		sc := mustScope(t, s)
		if got := sc.String(); got != s {
			t.Errorf("round trip of %q gave %q", s, got)
		}
	}
}
