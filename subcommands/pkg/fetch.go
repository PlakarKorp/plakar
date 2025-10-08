package pkg

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
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/utils"
)

func isRemote(name string) bool {
	return strings.HasPrefix(name, "https://") || strings.HasPrefix(name, "http://")
}

func isBase(name string) bool {
	return !filepath.IsAbs(name) && !strings.Contains(name, string(os.PathSeparator))
}

func openURL(ctx *appcontext.AppContext, path string) (io.ReadCloser, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", path, err)
	}

	token := ""
	if u.Hostname() == "plugins.plakar.io" && strings.HasPrefix(u.Path, "/kloset/pkg/") {
		token, _ = ctx.GetCookies().GetAuthToken()
		if token == "" {
			return nil, fmt.Errorf("login required for %q; run `plakar login` to authenticate", path)
		}
	}

	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("plakar/%s (%s/%s)", utils.VERSION, runtime.GOOS, runtime.GOARCH))
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	client := http.Client{}
	ctx.GetLogger().Info("fetching %s", path)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return resp.Body, nil
}

func getRecipe(ctx *appcontext.AppContext, name string, recipe *plugins.Recipe) error {
	var rd io.ReadCloser
	var err error

	fullpath := name

	remote := isRemote(fullpath)
	if !remote && isBase(fullpath) {
		u := *plugins.RecipeURL
		u.Path = path.Join(u.Path, fullpath)
		if !strings.HasPrefix(name, ".yaml") {
			u.Path += ".yaml"
		}
		fullpath = u.String()
		remote = true
	}

	if remote {
		rd, err = openURL(ctx, fullpath)
	} else {
		rd, err = os.Open(fullpath)
	}
	if err != nil {
		return fmt.Errorf("can't open %s: %w", name, err)
	}

	defer rd.Close()
	return recipe.Parse(rd)
}
