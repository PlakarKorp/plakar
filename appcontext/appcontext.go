package appcontext

import (
	"context"
	"errors"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/kcontext"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/PlakarKorp/plakar/signify"
)

type AppContext struct {
	*kcontext.KContext

	cookies     *cookies.Manager  `msgpack:"-"`
	pkgmgr      *pkg.Manager      `msgpack:"-"`
	pkgverifier *signify.Verifier `msgpack:"-"`
	Config      *config.Config    `msgpack:"-"`

	ConfigDir string
	secret    []byte

	StoreConfig map[string]string

	Quiet  bool
	Silent bool
}

func NewAppContext() *AppContext {
	return &AppContext{
		KContext: kcontext.NewKContext(),
	}
}

func NewAppContextFrom(ctx *AppContext) *AppContext {
	return &AppContext{
		KContext: kcontext.NewKContextFrom(ctx.GetInner()),

		cookies:     ctx.cookies,
		pkgmgr:      ctx.pkgmgr,
		pkgverifier: ctx.pkgverifier,
		ConfigDir:   ctx.ConfigDir,
	}
}

// XXX: This needs to go away progressively by migrating to AppContext.
func (c *AppContext) GetInner() *kcontext.KContext {
	return c.KContext
}

func (c *AppContext) ErrorCause(err error) error {
	if errors.Is(err, context.Canceled) {
		if cause := context.Cause(c.Context); cause != nil {
			return cause
		}
	}
	return err
}

func (c *AppContext) SetSecret(secret []byte) {
	c.secret = secret
}

func (c *AppContext) GetSecret() []byte {
	return c.secret
}

func (ctx *AppContext) ImporterOpts() *connectors.Options {
	return &connectors.Options{
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

func (ctx *AppContext) ExporterOpts() *connectors.Options {
	return &connectors.Options{
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

func (c *AppContext) SetPkgManager(pluginsManager *pkg.Manager) {
	c.pkgmgr = pluginsManager
}

func (c *AppContext) GetPkgManager() *pkg.Manager {
	return c.pkgmgr
}

func (c *AppContext) SetPkgVerifier(verifier *signify.Verifier) {
	c.pkgverifier = verifier
}

func (c *AppContext) GetPkgVerifier() *signify.Verifier {
	return c.pkgverifier
}

func (c *AppContext) ReloadConfig() error {
	cfg, err := utils.LoadConfig(c.ConfigDir)
	if err != nil {
		return err
	}
	c.Config = cfg
	return nil
}
