package appcontext

import (
	"context"
	"io"
	"os"

	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/encryption/keypair"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/google/uuid"
)

type AppContext struct {
	events  *events.Receiver `msgpack:"-"`
	cache   *caching.Manager `msgpack:"-"`
	logger  *logging.Logger  `msgpack:"-"`
	context context.Context  `msgpack:"-"`
	secret  []byte           `msgpack:"-"`
	Config  *config.Config   `msgpack:"-"`

	Stdout io.Writer `msgpack:"-"`
	Stderr io.Writer `msgpack:"-"`

	NumCPU      int
	Username    string
	HomeDir     string
	Hostname    string
	CommandLine string
	MachineID   string
	KeyFromFile string
	CacheDir    string
	KeyringDir  string

	OperatingSystem string
	Architecture    string
	ProcessID       int

	Client string

	CWD            string
	MaxConcurrency int

	Identity uuid.UUID
	Keypair  *keypair.KeyPair

	SnapshotID objects.MAC
}

func NewAppContext() *AppContext {
	return &AppContext{
		events:  events.New(),
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		context: context.Background(),
	}
}

func NewAppContextFrom(template *AppContext) *AppContext {
	ctx := NewAppContext()
	events := ctx.events
	*ctx = *template
	ctx.SetCache(template.GetCache())
	ctx.SetLogger(template.GetLogger())
	ctx.events = events
	return ctx
}

func (c *AppContext) Close() {
	c.events.Close()
}

func (c *AppContext) Events() *events.Receiver {
	return c.events
}

func (c *AppContext) SetCache(cacheManager *caching.Manager) {
	c.cache = cacheManager
}

func (c *AppContext) GetCache() *caching.Manager {
	return c.cache
}

func (c *AppContext) SetLogger(logger *logging.Logger) {
	c.logger = logger
}

func (c *AppContext) GetLogger() *logging.Logger {
	return c.logger
}

func (c *AppContext) SetContext(ctx context.Context) {
	c.context = ctx
}

func (c *AppContext) GetContext() context.Context {
	return c.context
}

func (c *AppContext) SetSecret(secret []byte) {
	c.secret = secret
}

func (c *AppContext) GetSecret() []byte {
	return c.secret
}
