package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PlakarKorp/pkg"
	"github.com/stretchr/testify/require"
)

// TestProxy drives the services-proxy endpoints. servicesProxy reads
// PLAKAR_SERVICE_ENDPOINT per request, so the endpoint-error cases can use
// t.Setenv against the shared mux instead of rebuilding it.
func TestProxy(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	t.Run("alerting get unauthorized", func(t *testing.T) {
		// No auth token in the cookie jar -> handler returns a 401 JSON body.
		w := get(t, mux, "/api/proxy/v1/account/services/alerting")
		require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())

		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, "authorization_error", resp["error"])
	})

	t.Run("alerting set unauthorized", func(t *testing.T) {
		body := `{"enabled":true,"email_report":true}`
		req, _ := http.NewRequest("PUT", "/api/proxy/v1/account/services/alerting", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code, "body=%s", w.Body.String())

		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, "authorization_error", resp["error"])
	})

	t.Run("integration path not implemented", func(t *testing.T) {
		w := get(t, mux, "/api/proxy/v1/integration/some-id/some/path")
		// Returns a plain error -> mapped to 500 by handleError.
		require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	})

	t.Run("bad endpoint", func(t *testing.T) {
		// An unparseable endpoint URL exercises the
		// `targetBase, err := url.Parse(...); if err != nil` branch.
		t.Setenv("PLAKAR_SERVICE_ENDPOINT", "://this is not a url")

		w := get(t, mux, "/api/proxy/v1/account/me")
		require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	})

	t.Run("unreachable", func(t *testing.T) {
		// A closed local port makes the outbound http.DefaultClient.Do fail with
		// connection-refused, exercising the `if err != nil` branch after it.
		// Hermetic: nothing listens on port 1.
		t.Setenv("PLAKAR_SERVICE_ENDPOINT", "http://127.0.0.1:1")

		w := get(t, mux, "/api/proxy/v1/account/me")
		require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	})
}

// TestIntegrationMatchesSearch pins the search filter used by
// servicesGetIntegration: case-insensitive substring over Name OR
// DisplayName; empty needle matches everything.
func TestIntegrationMatchesSearch(t *testing.T) {
	cases := []struct {
		name, needle string
		it           pkg.Integration
		want         bool
	}{
		{"empty needle matches", "", pkg.Integration{Name: "aws-s3"}, true},
		{"name substring, lowercase", "aws", pkg.Integration{Name: "aws-s3"}, true},
		{"name substring, mixed case in field", "s3", pkg.Integration{Name: "aws-S3", DisplayName: "Amazon S3"}, true},
		{"display name substring", "amazon", pkg.Integration{Name: "aws-s3", DisplayName: "Amazon S3"}, true},
		{"no match", "gcp", pkg.Integration{Name: "aws-s3", DisplayName: "Amazon S3"}, false},
		{"empty display name doesn't panic", "aws", pkg.Integration{Name: "aws-s3", DisplayName: ""}, true},
		{"empty name doesn't panic", "amazon", pkg.Integration{Name: "", DisplayName: "Amazon S3"}, true},
		{"unicode passthrough", "π", pkg.Integration{Name: "geo-π-service"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := integrationMatchesSearch(&tc.it, tc.needle)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestIntegrationMatchesInstallationStatus pins the installation_status
// filter used by servicesGetIntegration: empty filter matches
// everything; "installed"/"not-installed" match on Installation.Status;
// "upgradable" requires installed AND Installation.Version !=
// LatestVersion (plain string inequality, matches plakman); any unknown
// value matches nothing.
func TestIntegrationMatchesInstallationStatus(t *testing.T) {
	installed := func(ver, latest string) pkg.Integration {
		return pkg.Integration{
			Installation:  pkg.IntegrationInstallation{Status: "installed", Version: ver},
			LatestVersion: latest,
		}
	}
	notInstalled := func(latest string) pkg.Integration {
		return pkg.Integration{
			Installation:  pkg.IntegrationInstallation{Status: "not-installed"},
			LatestVersion: latest,
		}
	}

	cases := []struct {
		name   string
		filter string
		it     pkg.Integration
		want   bool
	}{
		{"empty filter matches installed", "", installed("1.0.0", "1.0.0"), true},
		{"empty filter matches not-installed", "", notInstalled("1.0.0"), true},

		{"installed matches installed row", "installed", installed("1.0.0", "1.0.0"), true},
		{"installed rejects not-installed row", "installed", notInstalled("1.0.0"), false},
		{"not-installed matches not-installed row", "not-installed", notInstalled("1.0.0"), true},
		{"not-installed rejects installed row", "not-installed", installed("1.0.0", "1.0.0"), false},

		{"upgradable when installed version differs", "upgradable", installed("1.0.0", "1.1.0"), true},
		{"upgradable rejects matching versions", "upgradable", installed("1.0.0", "1.0.0"), false},
		{"upgradable rejects not-installed row", "upgradable", notInstalled("1.1.0"), false},
		{"upgradable with empty installed version", "upgradable", installed("", "1.0.0"), true},
		{"upgradable with both versions empty", "upgradable", installed("", ""), false},

		{"unknown filter rejects installed", "garbage", installed("1.0.0", "1.0.0"), false},
		{"unknown filter rejects not-installed", "garbage", notInstalled("1.0.0"), false},
		{"case-sensitive: 'Installed' rejects all", "Installed", installed("1.0.0", "1.0.0"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := integrationMatchesInstallationStatus(&tc.it, tc.filter)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestIntegrationMatchesTypes pins the multi-value type filter used by
// servicesGetIntegration: empty filter matches everything; any-of
// semantics — an integration matches if it satisfies AT LEAST ONE of
// the requested Types bools; unknown values are ignored (match nothing
// on their own).
func TestIntegrationMatchesTypes(t *testing.T) {
	mk := func(storage, source, destination, provider bool) pkg.Integration {
		return pkg.Integration{
			Types: pkg.IntegrationTypes{
				Storage:     storage,
				Source:      source,
				Destination: destination,
				Provider:    provider,
			},
		}
	}

	storageOnly := mk(true, false, false, false)
	sourceOnly := mk(false, true, false, false)
	destOnly := mk(false, false, true, false)
	providerOnly := mk(false, false, false, true)
	sourceAndDest := mk(false, true, true, false)
	nothing := mk(false, false, false, false)

	cases := []struct {
		name   string
		filter []string
		it     pkg.Integration
		want   bool
	}{
		{"empty filter matches storage", nil, storageOnly, true},
		{"empty filter matches nothing-typed", nil, nothing, true},

		{"single value matches", []string{"storage"}, storageOnly, true},
		{"single value rejects wrong type", []string{"storage"}, sourceOnly, false},

		{"multi-value: first value matches", []string{"storage", "source"}, storageOnly, true},
		{"multi-value: second value matches", []string{"storage", "source"}, sourceOnly, true},
		{"multi-value: neither matches", []string{"storage", "source"}, destOnly, false},
		{"multi-value: both bools true = any-of", []string{"storage", "source"}, sourceAndDest, true}, // source matches
		{"all four values matches multi-typed", []string{"storage", "source", "destination", "provider"}, sourceAndDest, true},
		{"all four values rejects nothing-typed", []string{"storage", "source", "destination", "provider"}, nothing, false},

		{"provider matches provider row", []string{"provider"}, providerOnly, true},
		{"provider rejects non-provider row", []string{"provider"}, storageOnly, false},

		{"unknown value alone rejects", []string{"garbage"}, storageOnly, false},
		{"unknown mixed with valid still ORs", []string{"garbage", "storage"}, storageOnly, true},
		{"case-sensitive: 'Storage' rejects", []string{"Storage"}, storageOnly, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := integrationMatchesTypes(&tc.it, tc.filter)
			require.Equal(t, tc.want, got)
		})
	}
}

// TestIntegrationMatchesTag pins the tag filter used by
// servicesGetIntegration: empty tag matches everything; a non-empty tag
// must appear in the integration's Tags slice (case-sensitive
// slices.Contains match, unchanged from the pkg-side filter it
// replaces).
func TestIntegrationMatchesTag(t *testing.T) {
	withTags := func(tags ...string) pkg.Integration {
		return pkg.Integration{Tags: tags}
	}

	cases := []struct {
		name string
		tag  string
		it   pkg.Integration
		want bool
	}{
		{"empty tag matches everything", "", withTags(), true},
		{"empty tag matches integration with tags", "", withTags("backup", "daily"), true},
		{"exact tag match", "backup", withTags("backup"), true},
		{"one of multiple tags matches", "daily", withTags("backup", "daily"), true},
		{"no match", "weekly", withTags("backup", "daily"), false},
		{"case-sensitive: 'Backup' rejects", "Backup", withTags("backup"), false},
		{"empty Tags slice rejects non-empty tag", "backup", withTags(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := integrationMatchesTag(&tc.it, tc.tag)
			require.Equal(t, tc.want, got)
		})
	}
}
