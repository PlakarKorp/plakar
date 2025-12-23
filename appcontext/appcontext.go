package appcontext

import (
	"github.com/PlakarKorp/kloset/kcontext"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/utils"
)

type AppContext struct {
	*kcontext.KContext

	cookies *cookies.Manager `msgpack:"-"`
	plugins *plugins.Manager `msgpack:"-"`
	Config  *config.Config   `msgpack:"-"`

	ConfigDir string
	secret    []byte

	StoreConfig map[string]string
}

func NewAppContext() *AppContext {
	return &AppContext{
		KContext: kcontext.NewKContext(),
	}
}

func NewAppContextFrom(ctx *AppContext) *AppContext {
	return &AppContext{
		KContext: kcontext.NewKContextFrom(ctx.GetInner()),

		cookies:   ctx.cookies,
		plugins:   ctx.plugins,
		ConfigDir: ctx.ConfigDir,
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

func (c *AppContext) SetCookies(cacheManager *cookies.Manager) {
	c.cookies = cacheManager
}

func (c *AppContext) GetCookies() *cookies.Manager {
	return c.cookies
}

func (c *AppContext) SetPlugins(pluginsManager *plugins.Manager) {
	c.plugins = pluginsManager
}

func (c *AppContext) GetPlugins() *plugins.Manager {
	return c.plugins
}

func (c *AppContext) ReloadConfig() error {
	cfg, err := utils.LoadConfig(c.ConfigDir)
	if err != nil {
		return err
	}
	c.Config = cfg
	return nil
}
