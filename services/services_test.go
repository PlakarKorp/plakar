package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func newTestConnector(t *testing.T, endpoint string) *ServiceConnector {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.Client = "plakar-test/1.0.0"
	ctx.OperatingSystem = "testos"
	ctx.Architecture = "testarch"
	sc := NewServiceConnector(ctx, "tok")
	sc.endpoint = endpoint
	return sc
}

func TestNewServiceConnector(t *testing.T) {
	ctx := appcontext.NewAppContext()
	sc := NewServiceConnector(ctx, "abc")
	require.NotNil(t, sc)
	require.Equal(t, "abc", sc.authToken)
	require.Equal(t, SERVICE_ENDPOINT, sc.endpoint)
	require.Same(t, ctx, sc.appCtx)
}

func TestValidateConfigAlwaysNil(t *testing.T) {
	sd := &ServiceDescription{Name: "alerting"}
	require.NoError(t, sd.ValidateConfig(nil))
	require.NoError(t, sd.ValidateConfig(map[string]string{"k": "v"}))
}

func TestGetServiceList(t *testing.T) {
	want := []ServiceDescription{
		{Name: "alerting", DisplayName: "Alerting"},
		{Name: "backup", DisplayName: "Backup"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/account/services", r.URL.Path)
		require.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	got, err := sc.GetServiceList()
	require.NoError(t, err)
	require.Equal(t, want, got)

	// second call must come from cache (server would still work, but verify identity)
	got2, err := sc.GetServiceList()
	require.NoError(t, err)
	require.Equal(t, want, got2)
}

func TestGetServiceListCachedSkipsHTTP(t *testing.T) {
	sc := newTestConnector(t, "http://127.0.0.1:0")
	sc.servicesList = []ServiceDescription{{Name: "x"}}
	got, err := sc.GetServiceList()
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestGetServiceListBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	_, err := sc.GetServiceList()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get service list")
}

func TestGetServiceListBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	_, err := sc.GetServiceList()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal response")
}

func TestGetServiceListBadURL(t *testing.T) {
	// control characters in the URL cause http.NewRequest to fail.
	sc := newTestConnector(t, "http://\x7f")
	_, err := sc.GetServiceList()
	require.Error(t, err)
}

func TestGetServiceListConnRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing listening now
	sc := newTestConnector(t, url)
	_, err := sc.GetServiceList()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get service list")
}

func TestValidateServiceConfiguration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]ServiceDescription{{Name: "alerting"}})
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	require.NoError(t, sc.ValidateServiceConfiguration("alerting", map[string]string{}))

	err := sc.ValidateServiceConfiguration("missing", map[string]string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "service not found")
}

func TestValidateServiceConfigurationListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	err := sc.ValidateServiceConfiguration("alerting", nil)
	require.Error(t, err)
}

func TestGetServiceStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/account/services/alerting", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]bool{"enabled": true})
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	enabled, err := sc.GetServiceStatus("alerting")
	require.NoError(t, err)
	require.True(t, enabled)
}

func TestGetServiceStatusDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"enabled":false}`))
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	enabled, err := sc.GetServiceStatus("alerting")
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestGetServiceStatusBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusForbidden)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	_, err := sc.GetServiceStatus("alerting")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get service status")
}

func TestGetServiceStatusBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("garbage"))
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	_, err := sc.GetServiceStatus("alerting")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal response")
}

func TestGetServiceStatusBadURL(t *testing.T) {
	sc := newTestConnector(t, "http://\x7f")
	_, err := sc.GetServiceStatus("alerting")
	require.Error(t, err)
}

func TestGetServiceStatusNoToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Empty(t, r.Header.Get("Authorization"))
		w.Write([]byte(`{"enabled":true}`))
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	sc.authToken = ""
	enabled, err := sc.GetServiceStatus("alerting")
	require.NoError(t, err)
	require.True(t, enabled)
}

func TestSetServiceStatus(t *testing.T) {
	var got struct {
		Enabled bool `json:"enabled"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "PUT", r.Method)
		require.Equal(t, "/v1/account/services/alerting", r.URL.Path)
		json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	require.NoError(t, sc.SetServiceStatus("alerting", true))
	require.True(t, got.Enabled)
}

func TestSetServiceStatusBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusBadRequest)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	err := sc.SetServiceStatus("alerting", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to set service status")
}

func TestSetServiceStatusBadURL(t *testing.T) {
	sc := newTestConnector(t, "http://\x7f")
	err := sc.SetServiceStatus("alerting", true)
	require.Error(t, err)
}

func TestGetServiceConfiguration(t *testing.T) {
	want := map[string]string{"foo": "bar"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/account/services/alerting/configuration", r.URL.Path)
		json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	got, err := sc.GetServiceConfiguration("alerting")
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestGetServiceConfigurationBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusInternalServerError)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	_, err := sc.GetServiceConfiguration("alerting")
	require.Error(t, err)
}

func TestGetServiceConfigurationBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("nope"))
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	_, err := sc.GetServiceConfiguration("alerting")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal response")
}

func TestGetServiceConfigurationBadURL(t *testing.T) {
	sc := newTestConnector(t, "http://\x7f")
	_, err := sc.GetServiceConfiguration("alerting")
	require.Error(t, err)
}

func TestSetServiceConfiguration(t *testing.T) {
	var got map[string]string
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// first call: ValidateServiceConfiguration -> getServicesList
		if r.URL.Path == "/v1/account/services" {
			json.NewEncoder(w).Encode([]ServiceDescription{{Name: "alerting"}})
			return
		}
		require.Equal(t, "PUT", r.Method)
		require.Equal(t, "/v1/account/services/alerting/configuration", r.URL.Path)
		json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	cfg := map[string]string{"k": "v"}
	require.NoError(t, sc.SetServiceConfiguration("alerting", cfg))
	require.Equal(t, cfg, got)
}

func TestSetServiceConfigurationInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// service list does not contain "alerting" -> validation fails
		json.NewEncoder(w).Encode([]ServiceDescription{{Name: "other"}})
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	err := sc.SetServiceConfiguration("alerting", map[string]string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid configuration")
}

func TestSetServiceConfigurationPutBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/account/services" {
			json.NewEncoder(w).Encode([]ServiceDescription{{Name: "alerting"}})
			return
		}
		http.Error(w, "no", http.StatusBadRequest)
	}))
	defer srv.Close()

	sc := newTestConnector(t, srv.URL)
	err := sc.SetServiceConfiguration("alerting", map[string]string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to set service status")
}
