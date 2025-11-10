package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/PlakarKorp/plakar/appcontext"
	//"github.com/santhosh-tekuri/jsonschema/v6"
	errorspkg "github.com/PlakarKorp/plakar/internal/errors"
)

const SERVICE_ENDPOINT = "https://api.plakar.io"

type ServiceConnector struct {
	appCtx    *appcontext.AppContext
	authToken string
	endpoint  string

	servicesList []ServiceDescription
}

func NewServiceConnector(ctx *appcontext.AppContext, authToken string) *ServiceConnector {
	sc := &ServiceConnector{
		appCtx:    ctx,
		authToken: authToken,
		endpoint:  SERVICE_ENDPOINT,
	}
	registerServiceLogger(ctx.GetLogger())
	return sc
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
		return nil, errorspkg.Wrap(ErrBuildRequest, err, "failed to create request",
			errorspkg.WithContext("url", url))
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
		return nil, errorspkg.Wrap(ErrDoRequest, err, "failed to get service list",
			errorspkg.WithContext("url", url))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errorspkg.New(ErrUnexpectedStatus, "failed to get service list",
			errorspkg.WithContext("url", url),
			errorspkg.WithContext("status", resp.Status))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorspkg.Wrap(ErrReadBody, err, "failed to read service list",
			errorspkg.WithContext("url", url))
	}

	var res []ServiceDescription
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, errorspkg.Wrap(ErrDecodeResponse, err, "failed to decode service list",
			errorspkg.WithContext("url", url))
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
			if err := svc.ValidateConfig(config); err != nil {
				return errorspkg.Wrap(ErrValidateConfig, err, "invalid service configuration",
					errorspkg.WithContext("service", name))
			}
			return nil
		}
	}

	return errorspkg.New(ErrServiceNotFound, "service not found", errorspkg.WithContext("service", name))
}

func (sc *ServiceConnector) GetServiceList() ([]ServiceDescription, error) {
	return sc.getServicesList()
}

func (sc *ServiceConnector) GetServiceStatus(name string) (bool, error) {
	uri := fmt.Sprintf("/v1/account/services/%s", name)
	url := fmt.Sprintf("%s%s", sc.endpoint, uri)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, errorspkg.Wrap(ErrBuildRequest, err, "failed to create request",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
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
		return false, errorspkg.Wrap(ErrDoRequest, err, "failed to get service status",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, errorspkg.New(ErrUnexpectedStatus, "failed to get service status",
			errorspkg.WithContext("url", url),
			errorspkg.WithContext("service", name),
			errorspkg.WithContext("status", resp.Status))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, errorspkg.Wrap(ErrReadBody, err, "failed to read service status",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
	}

	var response struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return false, errorspkg.Wrap(ErrDecodeResponse, err, "failed to decode service status",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
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
		return errorspkg.Wrap(ErrEncodeRequest, err, "failed to marshal service status body",
			errorspkg.WithContext("service", name), errorspkg.WithContext("enabled", enabled))
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return errorspkg.Wrap(ErrBuildRequest, err, "failed to create request",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
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
		return errorspkg.Wrap(ErrDoRequest, err, "failed to set service status",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name), errorspkg.WithContext("enabled", enabled))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return errorspkg.New(ErrUnexpectedStatus, "failed to set service status",
			errorspkg.WithContext("url", url),
			errorspkg.WithContext("service", name),
			errorspkg.WithContext("enabled", enabled),
			errorspkg.WithContext("status", resp.Status))
	}

	return nil
}

func (sc *ServiceConnector) GetServiceConfiguration(name string) (map[string]string, error) {
	uri := fmt.Sprintf("/v1/account/services/%s/configuration", name)
	url := fmt.Sprintf("%s%s", sc.endpoint, uri)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errorspkg.Wrap(ErrBuildRequest, err, "failed to create request",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
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
		return nil, errorspkg.Wrap(ErrDoRequest, err, "failed to get service configuration",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errorspkg.New(ErrUnexpectedStatus, "failed to get service configuration",
			errorspkg.WithContext("url", url),
			errorspkg.WithContext("service", name),
			errorspkg.WithContext("status", resp.Status))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorspkg.Wrap(ErrReadBody, err, "failed to read service configuration",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
	}

	var response map[string]string
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, errorspkg.Wrap(ErrDecodeResponse, err, "failed to decode service configuration",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
	}

	return response, nil
}

func (sc *ServiceConnector) SetServiceConfiguration(name string, configuration map[string]string) error {

	if err := sc.ValidateServiceConfiguration(name, configuration); err != nil {
		return err
	}

	uri := fmt.Sprintf("/v1/account/services/%s/configuration", name)
	url := fmt.Sprintf("%s%s", sc.endpoint, uri)

	bodyBytes, err := json.Marshal(configuration)
	if err != nil {
		return errorspkg.Wrap(ErrEncodeRequest, err, "failed to marshal service configuration",
			errorspkg.WithContext("service", name))
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return errorspkg.Wrap(ErrBuildRequest, err, "failed to create request",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
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
		return errorspkg.Wrap(ErrDoRequest, err, "failed to set service configuration",
			errorspkg.WithContext("url", url), errorspkg.WithContext("service", name))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return errorspkg.New(ErrUnexpectedStatus, "failed to set service configuration",
			errorspkg.WithContext("url", url),
			errorspkg.WithContext("service", name),
			errorspkg.WithContext("status", resp.Status))
	}

	return nil
}
