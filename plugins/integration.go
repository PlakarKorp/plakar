package plugins

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	"github.com/PlakarKorp/plakar/utils"
)

type IntegrationInstallation struct {
	Status    string `json:"status"`
	Version   string `json:"version,omitempty"`
	Available bool   `json:"available"`
}

type IntegrationTypes struct {
	Storage     bool `json:"storage"`
	Source      bool `json:"source"`
	Destination bool `json:"destination"`
}

type Integration struct {
	Id            string           `json:"id"`
	Name          string           `json:"name"`
	DisplayName   string           `json:"display_name"`
	Description   string           `json:"description"`
	Homepage      string           `json:"homepage"`
	Repository    string           `json:"repository"`
	License       string           `json:"license"`
	Tags          []string         `json:"tags"`
	APIVersion    string           `json:"api_version"`
	LatestVersion string           `json:"latest_version"`
	Types         IntegrationTypes `json:"types"`

	Documentation string `json:"documentation"` // README.md
	Icon          string `json:"icon"`          // assets/icon.{png,svg}
	Featured      string `json:"featured"`      // assets/featured.{png,svg}

	Installation IntegrationInstallation `json:"installation"`
}

type IntegrationFilter struct {
	Type   string
	Tag    string
	Status string
}

type IntegrationIndex struct {
	Integrations []Integration `json:"integrations"`
}

func fetchIntegrationList() ([]Integration, error) {
	var index IntegrationIndex

	req, err := http.NewRequest("GET", "https://api.plakar.io/v1/integrations/list.json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("plakar/%s (%s/%s)", utils.VERSION, runtime.GOOS, runtime.GOARCH))
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&index)
	if err != nil {
		return nil, err
	}

	return index.Integrations, nil
}
