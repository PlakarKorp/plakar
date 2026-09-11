package services

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

// TestNetcovNewServiceConnectorHonorsAPIURL covers the PLAKAR_API_URL override
// branch of NewServiceConnector (otherwise only reachable via the env), and
// proves the override actually routes network calls to the local server.
//
// Uses t.Setenv, so no t.Parallel().
func TestNetcovNewServiceConnectorHonorsAPIURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/account/services", r.URL.Path)
		_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
	}))
	defer srv.Close()

	t.Setenv("PLAKAR_API_URL", srv.URL)

	ctx := appcontext.NewAppContext()
	ctx.Client = "plakar-test"
	ctx.OperatingSystem = "testos"
	ctx.Architecture = "testarch"

	sc := NewServiceConnector(ctx, "tok")
	require.Equal(t, srv.URL, sc.endpoint, "PLAKAR_API_URL must override the default endpoint")

	list, err := sc.GetServiceList()
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "alerting", list[0].Name)
}

// TestNetcovNewServiceConnectorEmptyAPIURLUsesDefault confirms that an empty
// PLAKAR_API_URL leaves the default endpoint in place (the override is skipped).
func TestNetcovNewServiceConnectorEmptyAPIURLUsesDefault(t *testing.T) {
	t.Setenv("PLAKAR_API_URL", "")

	ctx := appcontext.NewAppContext()
	sc := NewServiceConnector(ctx, "tok")
	require.Equal(t, SERVICE_ENDPOINT, sc.endpoint)
}
