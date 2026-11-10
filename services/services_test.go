package services

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
)

func newTestConnector(t *testing.T, handler http.Handler, token string) (*ServiceConnector, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	ctx := appcontext.NewAppContext()
	ctx.Client = "plakar-test"
	ctx.OperatingSystem = "testos"
	ctx.Architecture = "testarch"
	sc := NewServiceConnector(ctx, token)
	sc.endpoint = srv.URL
	return sc, srv
}

func TestNewServiceConnectorDefaults(t *testing.T) {
	ctx := appcontext.NewAppContext()
	sc := NewServiceConnector(ctx, "token")
	if sc.endpoint != SERVICE_ENDPOINT {
		t.Fatalf("endpoint = %q, want %q", sc.endpoint, SERVICE_ENDPOINT)
	}
	if sc.authToken != "token" {
		t.Fatalf("authToken = %q", sc.authToken)
	}
	if sc.appCtx != ctx {
		t.Fatal("appCtx not set")
	}
}

func TestValidateConfigStub(t *testing.T) {
	// ValidateConfig is currently a no-op stub. Keep coverage on it so we
	// notice when the real implementation is enabled.
	sd := &ServiceDescription{Name: "x"}
	if err := sd.ValidateConfig("anything"); err != nil {
		t.Fatalf("ValidateConfig returned %v, want nil", err)
	}
}

func TestGetServicesListSendsAuthAndUserAgent(t *testing.T) {
	var gotAuth, gotUA, gotAccept string
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/account/services" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `[{"name":"alerting","display_name":"Alerting","config_schema":{}}]`)
	}), "secret")

	list, err := sc.GetServiceList()
	if err != nil {
		t.Fatalf("GetServiceList: %v", err)
	}
	if len(list) != 1 || list[0].Name != "alerting" || list[0].DisplayName != "Alerting" {
		t.Fatalf("unexpected list: %+v", list)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Bearer secret")
	}
	if !strings.Contains(gotUA, "plakar-test") || !strings.Contains(gotUA, "testos") || !strings.Contains(gotUA, "testarch") {
		t.Fatalf("User-Agent = %q", gotUA)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept = %q", gotAccept)
	}
}

func TestGetServicesListNoAuthHeaderWhenTokenEmpty(t *testing.T) {
	var seenAuth string
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `[]`)
	}), "")

	if _, err := sc.GetServiceList(); err != nil {
		t.Fatalf("GetServiceList: %v", err)
	}
	if seenAuth != "" {
		t.Fatalf("expected no Authorization header, got %q", seenAuth)
	}
}

func TestGetServicesListCachesResult(t *testing.T) {
	calls := 0
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
	}), "tok")

	if _, err := sc.GetServiceList(); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := sc.GetServiceList(); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if calls != 1 {
		t.Fatalf("server hit %d times, want 1 (result should be cached)", calls)
	}
}

func TestGetServicesListNon200(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}), "tok")
	_, err := sc.GetServiceList()
	if err == nil {
		t.Fatal("expected error on 401, got nil")
	}
	if !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("err = %v, want ErrLoginRequired", err)
	}
}

func TestGetServicesListBadJSON(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `not json`)
	}), "tok")
	if _, err := sc.GetServiceList(); err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

func TestGetServiceStatus(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account/services/alerting" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("unexpected method %q", r.Method)
		}
		_, _ = io.WriteString(w, `{"enabled":true}`)
	}), "tok")

	enabled, err := sc.GetServiceStatus("alerting")
	if err != nil {
		t.Fatalf("GetServiceStatus: %v", err)
	}
	if !enabled {
		t.Fatal("expected enabled=true")
	}
}

func TestGetServiceStatusNon200(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusInternalServerError)
	}), "tok")
	if _, err := sc.GetServiceStatus("alerting"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetServiceStatusBadJSON(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `garbage`)
	}), "tok")
	if _, err := sc.GetServiceStatus("alerting"); err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

func TestSetServiceStatusEncodesPayloadAsPUT(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody struct {
		Enabled bool `json:"enabled"`
	}
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}), "tok")

	if err := sc.SetServiceStatus("alerting", true); err != nil {
		t.Fatalf("SetServiceStatus: %v", err)
	}
	if gotMethod != "PUT" {
		t.Fatalf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/v1/account/services/alerting" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if !gotBody.Enabled {
		t.Fatal("body.Enabled = false, want true")
	}
}

func TestSetServiceStatusAcceptsBoth200And204(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusNoContent} {
		sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}), "tok")
		if err := sc.SetServiceStatus("alerting", false); err != nil {
			t.Fatalf("status %d: SetServiceStatus = %v", code, err)
		}
	}
}

func TestSetServiceStatusNon2xx(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}), "tok")
	if err := sc.SetServiceStatus("alerting", true); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetServiceConfiguration(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account/services/alerting/configuration" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"webhook":"https://example.test"}`)
	}), "tok")
	cfg, err := sc.GetServiceConfiguration("alerting")
	if err != nil {
		t.Fatalf("GetServiceConfiguration: %v", err)
	}
	if cfg["webhook"] != "https://example.test" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestGetServiceConfigurationNon200(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "x", http.StatusForbidden)
	}), "tok")
	if _, err := sc.GetServiceConfiguration("alerting"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetServiceConfigurationBadJSON(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `nope`)
	}), "tok")
	if _, err := sc.GetServiceConfiguration("alerting"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSetServiceConfigurationRoundtrip(t *testing.T) {
	// SetServiceConfiguration first calls getServicesList to validate, then PUTs.
	requests := 0
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/v1/account/services":
			_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
		case "/v1/account/services/alerting/configuration":
			if r.Method != "PUT" {
				t.Errorf("method = %q, want PUT", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var m map[string]string
			if err := json.Unmarshal(body, &m); err != nil {
				t.Errorf("body unmarshal: %v", err)
			}
			if m["k"] != "v" {
				t.Errorf("body = %+v", m)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}), "tok")

	if err := sc.SetServiceConfiguration("alerting", map[string]string{"k": "v"}); err != nil {
		t.Fatalf("SetServiceConfiguration: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 (services list + config PUT)", requests)
	}
}

func TestSetServiceConfigurationServerError(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/account/services" {
			_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
			return
		}
		http.Error(w, "x", http.StatusInternalServerError)
	}), "tok")
	if err := sc.SetServiceConfiguration("alerting", map[string]string{"k": "v"}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateServiceConfigurationUnknownService(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
	}), "tok")
	if err := sc.ValidateServiceConfiguration("unknown", nil); err == nil {
		t.Fatal("expected service-not-found error, got nil")
	}
}

func TestValidateServiceConfigurationKnownService(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
	}), "tok")
	// ValidateConfig is currently a no-op stub, so this should succeed.
	if err := sc.ValidateServiceConfiguration("alerting", nil); err != nil {
		t.Fatalf("ValidateServiceConfiguration: %v", err)
	}
}

// badRequestConnector returns a connector whose endpoint contains a control
// character, so http.NewRequest fails before any network call is made.
func badRequestConnector(t *testing.T) *ServiceConnector {
	t.Helper()
	ctx := appcontext.NewAppContext()
	sc := NewServiceConnector(ctx, "tok")
	sc.endpoint = "http://\x7f" // ASCII DEL makes the URL invalid for NewRequest
	return sc
}

func TestNewRequestErrorPropagates(t *testing.T) {
	if _, err := badRequestConnector(t).GetServiceList(); err == nil {
		t.Fatal("GetServiceList: expected NewRequest error, got nil")
	}
	if _, err := badRequestConnector(t).GetServiceStatus("x"); err == nil {
		t.Fatal("GetServiceStatus: expected NewRequest error, got nil")
	}
	if err := badRequestConnector(t).SetServiceStatus("x", true); err == nil {
		t.Fatal("SetServiceStatus: expected NewRequest error, got nil")
	}
	if _, err := badRequestConnector(t).GetServiceConfiguration("x"); err == nil {
		t.Fatal("GetServiceConfiguration: expected NewRequest error, got nil")
	}
}

func TestSetServiceConfigurationNewRequestError(t *testing.T) {
	// getServicesList must succeed (so validation passes) before the bad URL is
	// hit on the configuration PUT. Point the services-list lookup at a working
	// server but the configuration write at an invalid URL.
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
	}), "tok")
	// Prime the cached services list, then break the endpoint.
	if _, err := sc.GetServiceList(); err != nil {
		t.Fatalf("priming services list: %v", err)
	}
	sc.endpoint = "http://\x7f"
	if err := sc.SetServiceConfiguration("alerting", map[string]string{"k": "v"}); err == nil {
		t.Fatal("expected NewRequest error, got nil")
	}
}

func TestSetServiceConfigurationNetworkError(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `[{"name":"alerting"}]`)
	}), "tok")
	if _, err := sc.GetServiceList(); err != nil {
		t.Fatalf("priming services list: %v", err)
	}
	sc.endpoint = "http://127.0.0.1:0" // unreachable
	if err := sc.SetServiceConfiguration("alerting", map[string]string{"k": "v"}); err == nil {
		t.Fatal("expected network error, got nil")
	}
}

// shortBodyHandler claims a large Content-Length but writes fewer bytes and
// closes the connection, so io.ReadAll on the client fails with an unexpected
// EOF.
func shortBodyHandler(w http.ResponseWriter, r *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return
	}
	defer conn.Close()
	_, _ = bufrw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 100\r\n\r\nshort")
	_ = bufrw.Flush()
	// connection closes here with body shorter than Content-Length
}

func TestReadBodyErrorPropagates(t *testing.T) {
	sc, _ := newTestConnector(t, http.HandlerFunc(shortBodyHandler), "tok")

	if _, err := sc.GetServiceList(); err == nil {
		t.Fatal("GetServiceList: expected read-body error, got nil")
	}
	if _, err := sc.GetServiceStatus("x"); err == nil {
		t.Fatal("GetServiceStatus: expected read-body error, got nil")
	}
	if _, err := sc.GetServiceConfiguration("x"); err == nil {
		t.Fatal("GetServiceConfiguration: expected read-body error, got nil")
	}
}

func TestValidateServiceConfigurationListError(t *testing.T) {
	// getServicesList fails (server returns non-200), so the error must surface.
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}), "tok")
	if err := sc.ValidateServiceConfiguration("alerting", nil); err == nil {
		t.Fatal("expected getServicesList error to propagate, got nil")
	}
}

func TestSetServiceConfigurationValidationError(t *testing.T) {
	// SetServiceConfiguration first validates, which calls getServicesList; a
	// failing list lookup must abort with the wrapped "invalid configuration"
	// error before any write request is made.
	sc, _ := newTestConnector(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}), "tok")
	if err := sc.SetServiceConfiguration("alerting", map[string]string{"k": "v"}); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestRequestNetworkErrorPropagates(t *testing.T) {
	ctx := appcontext.NewAppContext()
	sc := NewServiceConnector(ctx, "tok")
	sc.endpoint = "http://127.0.0.1:0" // unreachable
	if _, err := sc.GetServiceList(); err == nil {
		t.Fatal("expected network error, got nil")
	}
	if _, err := sc.GetServiceStatus("x"); err == nil {
		t.Fatal("expected network error, got nil")
	}
	if err := sc.SetServiceStatus("x", true); err == nil {
		t.Fatal("expected network error, got nil")
	}
	if _, err := sc.GetServiceConfiguration("x"); err == nil {
		t.Fatal("expected network error, got nil")
	}
}
