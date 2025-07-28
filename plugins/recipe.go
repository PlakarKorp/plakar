package plugins

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"go.yaml.in/yaml/v3"
)

var recipeURL, _ = url.Parse("https://plugins.plakar.io/kloset/recipe/" + PLUGIN_API_VERSION + "/")

type Recipe struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
	Checksum   string `yaml:"checksum"`
}

func GetRecipe(ctx *appcontext.AppContext, name string, recipe *Recipe) error {
	var rd io.ReadCloser
	var err error

	fullpath := name

	remote := strings.HasPrefix(fullpath, "https://") || strings.HasPrefix(fullpath, "http://")
	if !remote && !filepath.IsAbs(fullpath) && !strings.Contains(fullpath, string(os.PathSeparator)) {
		u := *recipeURL
		u.Path = path.Join(u.Path, fullpath)
		if !strings.HasPrefix(name, ".yaml") {
			u.Path += ".yaml"
		}
		fullpath = u.String()
		remote = true
	}

	if fullpath == "-" {
		rd = io.NopCloser(ctx.Stdin)
	} else if remote {
		resp, err := http.Get(fullpath)
		if err != nil {
			return fmt.Errorf("can't fetch %s: %w", fullpath, err)
		}
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
		}
		rd = resp.Body
	} else {
		rd, err = os.Open(fullpath)
	}
	if err != nil {
		return fmt.Errorf("can't open %s: %w", recipe, err)
	}
	defer rd.Close()

	if err := yaml.NewDecoder(rd).Decode(recipe); err != nil {
		return fmt.Errorf("failed to parse the recipe %s: %w", name, err)
	}

	return nil
}

func (recipe *Recipe) PkgName() string {
	GOOS := runtime.GOOS
	GOARCH := runtime.GOARCH
	if goosEnv := os.Getenv("GOOS"); goosEnv != "" {
		GOOS = goosEnv
	}
	if goarchEnv := os.Getenv("GOARCH"); goarchEnv != "" {
		GOARCH = goarchEnv
	}

	return fmt.Sprintf("%s_%s_%s_%s.ptar", recipe.Name, recipe.Version, GOOS, GOARCH)
}
