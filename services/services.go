package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/PlakarKorp/plakar/appcontext"
	//"github.com/santhosh-tekuri/jsonschema/v6"
)

const SERVICE_ENDPOINT = "https://api.plakar.io"

type ServiceConnector struct {
	appCtx    *appcontext.AppContext
	authToken string
	endpoint  string

	servicesList []ServiceDescription
}

func NewServiceConnector(ctx *appcontext.AppContext, authToken string) *ServiceConnector {
	return &ServiceConnector{
		appCtx:    ctx,
		authToken: authToken,
		endpoint:  SERVICE_ENDPOINT,
	}
}

type ServiceDescription struct {
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	ConfigSchema map[string]any `json:"config_schema"`
}

func (sd *ServiceDescription) ValidateConfig(value any) error {
	// not working for now
	// c := jsonschema.NewCompiler()
	// if err := c.AddResource(sd.Name+".json", sd.ConfigSchema); err != nil {
	// 	return err
	// }
	// schema, err := c.Compile(sd.Name + ".json")
	// if err != nil {
	// 	return err
	// }
	// if err := schema.Validate(value); err != nil {
	// 	return err
	// }
	return nil
}

func (sc *ServiceConnector) getServicesList() ([]ServiceDescription, error) {
	if sc.servicesList != nil {
		return sc.servicesList, nil
	}

	url := fmt.Sprintf("%s%s", sc.endpoint, "/v1/account/services")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s (%s/%s)", sc.appCtx.Client, sc.appCtx.OperatingSystem, sc.appCtx.Architecture))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Charset", "utf-8")
	if sc.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+sc.authToken)
	}

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get service list: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get service list: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var res []ServiceDescription
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	sc.servicesList = res
	return res, nil
}

func (sc *ServiceConnector) ValidateServiceConfiguration(name string, config any) error {
	svcs, err := sc.getServicesList()
	if err != nil {
		return err
	}
	for _, svc := range svcs {
		if svc.Name == name {
			return svc.ValidateConfig(config)
		}
	}

	return fmt.Errorf("service not found")
}

func (sc *ServiceConnector) GetServiceList() ([]ServiceDescription, error) {
	return sc.getServicesList()
}

func (sc *ServiceConnector) GetServiceStatus(name string) (bool, error) {
	uri := fmt.Sprintf("/v1/account/services/%s", name)
	url := fmt.Sprintf("%s%s", sc.endpoint, uri)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s (%s/%s)", sc.appCtx.Client, sc.appCtx.OperatingSystem, sc.appCtx.Architecture))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Charset", "utf-8")

	if sc.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+sc.authToken)
	}

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to get service status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to get service status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %v", err)
	}

	var response struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return response.Enabled, nil
}

func (sc *ServiceConnector) SetServiceStatus(name string, enabled bool) error {
	uri := fmt.Sprintf("/v1/account/services/%s", name)
	url := fmt.Sprintf("%s%s", sc.endpoint, uri)

	var body = struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: enabled,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s (%s/%s)", sc.appCtx.Client, sc.appCtx.OperatingSystem, sc.appCtx.Architecture))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Charset", "utf-8")

	if sc.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+sc.authToken)
	}

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get service status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to set service status: %s", resp.Status)
	}

	return nil
}

func (sc *ServiceConnector) GetServiceConfiguration(name string) (map[string]string, error) {
	uri := fmt.Sprintf("/v1/account/services/%s/configuration", name)
	url := fmt.Sprintf("%s%s", sc.endpoint, uri)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s (%s/%s)", sc.appCtx.Client, sc.appCtx.OperatingSystem, sc.appCtx.Architecture))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Charset", "utf-8")

	if sc.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+sc.authToken)
	}

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get service status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get service status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var response map[string]string
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return response, nil
}

func (sc *ServiceConnector) SetServiceConfiguration(name string, configuration map[string]string) error {

	if err := sc.ValidateServiceConfiguration(name, configuration); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	uri := fmt.Sprintf("/v1/account/services/%s/configuration", name)
	url := fmt.Sprintf("%s%s", sc.endpoint, uri)

	bodyBytes, err := json.Marshal(configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s (%s/%s)", sc.appCtx.Client, sc.appCtx.OperatingSystem, sc.appCtx.Architecture))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Charset", "utf-8")

	if sc.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+sc.authToken)
	}

	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set service status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to set service status: %s", resp.Status)
	}

	return nil
}
