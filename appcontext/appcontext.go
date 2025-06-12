package appcontext

import (
	"github.com/PlakarKorp/kloset/kcontext"
	"github.com/PlakarKorp/kloset/snapshot/importer"
)

type AppContext struct {
	*kcontext.KContext
	secret []byte
}

func NewAppContext() *AppContext {
	return &AppContext{
		KContext: kcontext.NewKContext(),
	}
}

func NewAppContextFrom(ctx *AppContext) *AppContext {
	return &AppContext{
		KContext: kcontext.NewKContextFrom(ctx.GetInner()),
	}
}

// XXX: This needs to go away progressively by migrating to AppContext.
func (c *AppContext) GetInner() *kcontext.KContext {
	return c.KContext
}

func (c *AppContext) SetSecret(secret []byte) {
	c.secret = secret
}

func (c *AppContext) GetSecret() []byte {
	return c.secret
}

func (ctx *AppContext) ImporterOpts() *importer.Options {
	return &importer.Options{
		Hostname:        ctx.Hostname,
		OperatingSystem: ctx.OperatingSystem,
		Architecture:    ctx.Architecture,
		CWD:             ctx.CWD,
		MaxConcurrency:  ctx.MaxConcurrency,
		Stdin:           ctx.Stdin,
		Stdout:          ctx.Stdout,
		Stderr:          ctx.Stderr,
	}
}
