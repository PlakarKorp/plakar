package pkg

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
)

func getRecipe(ctx *appcontext.AppContext, name string, recipe *pkg.Recipe) error {
	switch {
	case strings.HasPrefix(name, "https://") || strings.HasPrefix(name, "http://"):
		res, err := http.Get(name)
		if err != nil {
			return err
		}
		if res.StatusCode != 200 {
			res.Body.Close()
			return fmt.Errorf("couldn't fetch recipe: serve failed with %s",
				res.Status)
		}
		defer res.Body.Close()
		return recipe.Parse(res.Body)

	case filepath.IsAbs(name) || strings.Contains(name, string(os.PathSeparator)):
		fp, err := os.Open(name)
		if err != nil {
			return fmt.Errorf("couldn't open %s: %w", name, err)
		}
		defer fp.Close()
		return recipe.Parse(fp)

	default:
		r, err := ctx.GetPkgManager().FetchRecipe(name, nil)
		if err != nil {
			return err
		}
		*recipe = *r
		return nil
	}
}
