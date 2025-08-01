package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/services"
)

var SERVICES_ENDPOINT = "https://api.plakar.io"

func (ui *uiserver) servicesProxy(w http.ResponseWriter, r *http.Request) error {
	// Define target service base URL
	serviceEndpoint := os.Getenv("PLAKAR_SERVICE_ENDPOINT")
	if serviceEndpoint == "" {
		serviceEndpoint = SERVICES_ENDPOINT
	}

	targetBase, err := url.Parse(serviceEndpoint)
	if err != nil {
		return err
	}

	// Construct target URL by preserving the path and query parameters
	targetURL := targetBase.ResolveReference(&url.URL{
		Path:     strings.TrimPrefix(r.URL.Path, "/api/proxy"),
		RawQuery: r.URL.RawQuery,
	})

	// Create new request to target
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		return err
	}

	// Copy headers from original request
	client := fmt.Sprintf("%s (%s/%s)",
		ui.ctx.Client,
		ui.ctx.OperatingSystem,
		ui.ctx.Architecture)

	authToken, _ := ui.ctx.GetCookies().GetAuthToken()
	if authToken != "" {
		req.Header.Add("Authorization", "Bearer "+authToken)
	}
	req.Header.Add("User-Agent", client)
	req.Header.Add("X-Real-IP", r.RemoteAddr)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	for name, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(name, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	return err
}

type AlertServiceConfiguration struct {
	Enabled     bool `json:"enabled"`
	EmailReport bool `json:"email_report"`
}

func (ui *uiserver) servicesGetAlertingServiceConfiguration(w http.ResponseWriter, r *http.Request) error {
	authToken, _ := ui.ctx.GetCookies().GetAuthToken()

	if authToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "authorization_error",
			"message": "Authorization required",
		})
		return nil
	}

	sc := services.NewServiceConnector(ui.ctx, authToken)
	enabled, err := sc.GetServiceStatus("alerting")
	if err != nil {
		return err
	}

	config, err := sc.GetServiceConfiguration("alerting")
	if err != nil {
		return err
	}

	var alertConfig AlertServiceConfiguration
	alertConfig.Enabled = enabled
	if emailReport, ok := config["report.email"]; ok {
		if emailReport == "true" {
			alertConfig.EmailReport = true
		} else {
			alertConfig.EmailReport = false
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(alertConfig)
}

func (ui *uiserver) servicesSetAlertingServiceConfiguration(w http.ResponseWriter, r *http.Request) error {
	authToken, _ := ui.ctx.GetCookies().GetAuthToken()

	if authToken == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "authorization_error",
			"message": "Authorization required",
		})
		return nil
	}

	var alertConfig AlertServiceConfiguration
	if err := json.NewDecoder(r.Body).Decode(&alertConfig); err != nil {
		return err
	}

	sc := services.NewServiceConnector(ui.ctx, authToken)

	err := sc.SetServiceStatus("alerting", alertConfig.Enabled)
	if err != nil {
		return err
	}

	err = sc.SetServiceConfiguration("alerting", map[string]string{
		"report.email": fmt.Sprintf("%t", alertConfig.EmailReport),
	})
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(alertConfig)
}

func (ui *uiserver) servicesGetIntegration(w http.ResponseWriter, r *http.Request) error {
	offset, err := QueryParamToInt64(r, "offset", 0, 0)
	if err != nil {
		return err
	}

	limit, err := QueryParamToInt64(r, "limit", 1, 50)
	if err != nil {
		return err
	}

	filterType, _, err := QueryParamToString(r, "type")
	if err != nil {
		return err
	}

	filterTag, _, err := QueryParamToString(r, "tag")
	if err != nil {
		return err
	}

	filterStatus, _, err := QueryParamToString(r, "status")
	if err != nil {
		return err
	}

	var res Items[plugins.Integration]
	res.Items = make([]plugins.Integration, 0)

	var i int64
	filter := plugins.IntegrationFilter{
		Type:   filterType,
		Tag:    filterTag,
		Status: filterStatus,
	}

	ui.reloadPlugins()

	ints, err := ui.ctx.GetPlugins().ListIntegrations(filter)
	if err != nil {
		return err
	}
	for _, int := range ints {
		res.Total += 1
		i += 1
		if i > offset {
			if i <= offset+limit {
				res.Items = append(res.Items, int)
			}
		}
	}

	return json.NewEncoder(w).Encode(res)
}

func (ui *uiserver) servicesGetIntegrationId(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	ui.reloadPlugins()

	var filter plugins.IntegrationFilter
	ints, err := ui.ctx.GetPlugins().ListIntegrations(filter)
	if err != nil {
		return err
	}

	for _, int := range ints {
		if int.Id == id {
			return json.NewEncoder(w).Encode(int)
		}
	}

	return fmt.Errorf("Not found")
}

func (ui *uiserver) servicesGetIntegrationPath(w http.ResponseWriter, r *http.Request) error {
	return fmt.Errorf("Not implemented")
}
