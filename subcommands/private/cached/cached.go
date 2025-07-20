package cached

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cached"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Cached{} },
		subcommands.BeforeRepositoryOpen,
		"private-cached")
}

type Cached struct {
	subcommands.SubcommandBase
}

func (cmd *Cached) Parse(ctx *appcontext.AppContext, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("bad usage")
	}

	return nil
}

func (cmd *Cached) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	err := cached.ListenAndServe("/tmp/2.0.0/cached.sock")
	if err != nil {
		return 1, err
	}

	return 0, nil
}
