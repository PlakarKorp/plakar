package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var SERVICES_ENDPOINT = "https://api.plakar.io"

func servicesProxy(w http.ResponseWriter, r *http.Request) error {
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
		lrepository.AppContext().Client,
		lrepository.AppContext().OperatingSystem,
		lrepository.AppContext().Architecture)

	configuration := lrepository.Configuration()
	if cache, err := lrepository.AppContext().GetCache().Repository(configuration.RepositoryID); err == nil {
		if authToken, err := cache.GetAuthToken(); err == nil && authToken != "" {
			req.Header.Add("Authorization", "Bearer "+authToken)
		}
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
