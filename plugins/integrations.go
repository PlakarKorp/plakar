package plugins

import (
	_ "embed"
	"encoding/json"
	"iter"
	"slices"

	"github.com/PlakarKorp/plakar/appcontext"
)

type IntegrationInstallation struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

type IntegrationTypes struct {
	Storage     bool `json:"storage"`
	Source      bool `json:"source"`
	Destination bool `json:"destination"`
}

type IntegrationInfo struct {
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

type IntegrationIndex struct {
	Integrations []IntegrationInfo `json:"integrations"`
}

//go:embed integrations.json
var integrationsData []byte

func IterIntegrations(ctx *appcontext.AppContext, filterType string, filterTag string) iter.Seq2[*IntegrationInfo, error] {
	return func(yield func(*IntegrationInfo, error) bool) {
		var index IntegrationIndex
		err := json.Unmarshal(integrationsData, &index)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, info := range index.Integrations {
			if filterType == "storage" && !info.Types.Storage {
				continue
			}
			if filterType == "source" && !info.Types.Source {
				continue
			}
			if filterType == "destination" && !info.Types.Destination {
				continue
			}
			if filterTag != "" && !slices.Contains(info.Tags, filterTag) {
				continue
			}
			// XXX Check if installed
			info.Installation.Status = "not-installed"

			if !yield(&info, nil) {
				return
			}
		}
	}
}
