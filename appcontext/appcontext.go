package appcontext

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/encryption/keypair"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/google/uuid"
)

type AppContext struct {
	events *events.Receiver `msgpack:"-"`
	cache  *caching.Manager `msgpack:"-"`
	logger *logging.Logger  `msgpack:"-"`
	secret []byte           `msgpack:"-"`
	Config *config.Config   `msgpack:"-"`

	Context context.Context    `msgpack:"-"`
	Cancel  context.CancelFunc `msgpack:"-"`

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
}

func NewAppContext() *AppContext {
	ctx, cancel := context.WithCancel(context.Background())

	return &AppContext{
		events:  events.New(),
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Context: ctx,
		Cancel:  cancel,
	}
}

func NewAppContextFrom(template *AppContext) *AppContext {
	ctx := *template
	ctx.events = events.New()
	ctx.Context, ctx.Cancel = context.WithCancel(template.Context)
	return &ctx
}

func (c *AppContext) Deadline() (time.Time, bool) {
	return c.Context.Deadline()
}

func (c *AppContext) Done() <-chan struct{} {
	return c.Context.Done()
}

func (c *AppContext) Err() error {
	return c.Context.Err()
}

func (c *AppContext) Value(key any) any {
	return c.Context.Value(key)
}

func (c *AppContext) Close() {
	c.events.Close()
	c.Cancel()
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

func (c *AppContext) SetSecret(secret []byte) {
	c.secret = secret
}

func (c *AppContext) GetSecret() []byte {
	return c.secret
}

func (c *AppContext) GetAuthToken(repository uuid.UUID) (string, error) {
	if cache, err := c.cache.Repository(repository); err != nil {
		return "", err
	} else if authToken, err := cache.GetAuthToken(); err != nil {
		return "", err
	} else {
		return authToken, nil
	}
}
