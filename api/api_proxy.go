package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/services"
)

var SERVICES_ENDPOINT = "https://api.plakar.io"

const integrationsCacheKey = "all"

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

func integrationMatchesSearch(it *pkg.Integration, needle string) bool {
	if needle == "" {
		return true
	}
	needle = strings.ToLower(needle)
	return strings.Contains(strings.ToLower(it.Name), needle) ||
		strings.Contains(strings.ToLower(it.DisplayName), needle)
}

func integrationMatchesInstallationStatus(it *pkg.Integration, filter string) bool {
	switch filter {
	case "":
		return true
	case "installed":
		return it.Installation.Status == "installed"
	case "not-installed":
		return it.Installation.Status == "not-installed"
	case "upgradable":
		return it.Installation.Status == "installed" &&
			it.Installation.Version != it.LatestVersion
	default:
		return false
	}
}

func integrationMatchesTag(it *pkg.Integration, tag string) bool {
	if tag == "" {
		return true
	}
	return slices.Contains(it.Tags, tag)
}

func integrationMatchesTypes(it *pkg.Integration, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, t := range filter {
		switch t {
		case "storage":
			if it.Types.Storage {
				return true
			}
		case "source":
			if it.Types.Source {
				return true
			}
		case "destination":
			if it.Types.Destination {
				return true
			}
		case "provider":
			if it.Types.Provider {
				return true
			}
		}
	}
	return false
}

func (ui *uiserver) cachedIntegrations() ([]*pkg.Integration, error) {
	if cached, ok := ui.integrationsCache.Get(integrationsCacheKey); ok {
		return cached, nil
	}
	integrations, err := ui.ctx.GetPkgManager().Query(nil)
	if err != nil {
		return nil, err
	}
	ui.integrationsCache.Add(integrationsCacheKey, integrations)
	return integrations, nil
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

	filterTypes := QueryParamToStrings(r, "type")

	filterTag, _, err := QueryParamToString(r, "tag")
	if err != nil {
		return err
	}

	// installation_status is the canonical param; `status` is kept as a
	// legacy alias. If both are sent, installation_status wins.
	filterInstallationStatus, _, err := QueryParamToString(r, "installation_status")
	if err != nil {
		return err
	}
	if filterInstallationStatus == "" {
		filterInstallationStatus, _, err = QueryParamToString(r, "status")
		if err != nil {
			return err
		}
	}

	search, _, err := QueryParamToString(r, "search")
	if err != nil {
		return err
	}

	var res Items[pkg.Integration]
	res.Items = make([]pkg.Integration, 0)

	integrations, err := ui.cachedIntegrations()
	if err != nil {
		return err
	}

	var i int64
	for _, integration := range integrations {
		if !integrationMatchesSearch(integration, search) {
			continue
		}
		if !integrationMatchesInstallationStatus(integration, filterInstallationStatus) {
			continue
		}
		if !integrationMatchesTypes(integration, filterTypes) {
			continue
		}
		if !integrationMatchesTag(integration, filterTag) {
			continue
		}

		res.Total++
		if i >= offset && i < offset+limit {
			res.Items = append(res.Items, *integration)
		}
		i++
	}

	return json.NewEncoder(w).Encode(res)
}

func (ui *uiserver) servicesGetIntegrationId(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	integrations, err := ui.cachedIntegrations()
	if err != nil {
		return err
	}

	for _, int := range integrations {
		if int.Id == id {
			return json.NewEncoder(w).Encode(int)
		}
	}

	return fmt.Errorf("not found")
}

func (ui *uiserver) servicesGetIntegrationPath(w http.ResponseWriter, r *http.Request) error {
	return fmt.Errorf("not implemented")
}
