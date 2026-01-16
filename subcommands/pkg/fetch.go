package pkg

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
)

func isRemote(name string) bool {
	return strings.HasPrefix(name, "https://") || strings.HasPrefix(name, "http://")
}

func isBase(name string) bool {
	return !filepath.IsAbs(name) && !strings.Contains(name, string(os.PathSeparator))
}

func getRecipe(ctx *appcontext.AppContext, name string, recipe *pkg.Recipe) error {
	var rd io.ReadCloser
	var err error

	fullpath := name

	remote := isRemote(fullpath)
	if !remote && isBase(fullpath) {
		//u := *plugins.RecipeURL
		var u url.URL
		u.Path = path.Join(u.Path, fullpath)
		if !strings.HasPrefix(name, ".yaml") {
			u.Path += ".yaml"
		}
		fullpath = u.String()
		remote = true
	}

	if remote {
		//rd, err = openURL(ctx, fullpath)
	} else {
		rd, err = os.Open(fullpath)
	}
	if err != nil {
		return fmt.Errorf("can't open %s: %w", name, err)
	}

	defer rd.Close()
	return recipe.Parse(rd)
}
