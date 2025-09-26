package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

var ErrCantUnlock = errors.New("failed to unlock repository")

// RepositoryManager owns the repository and store lifecycle.
type RepositoryManager struct {
	store       storage.Store
	repository  *repository.Repository
	openedStore bool
}

func (m *RepositoryManager) Store() storage.Store               { return m.store }
func (m *RepositoryManager) Repository() *repository.Repository { return m.repository }

// RepositoryStage opens the repository according to command requirements.
type RepositoryStage struct{}

// NewRepositoryStage creates a repository stage.
func NewRepositoryStage() *RepositoryStage {
	return &RepositoryStage{}
}

func (s *RepositoryStage) Name() string { return "repository" }

func (s *RepositoryStage) Execute(ctx *ConfigContext) error {
	manager := &RepositoryManager{}
	ctx.Repository = manager

	cmd := ctx.Command
	if cmd == nil {
		return fmt.Errorf("bootstrap repository stage requires a command")
	}

	flags := cmd.GetFlags()

	if flags&subcommands.BeforeRepositoryOpen != 0 {
		if ctx.AtSyntax {
			return fmt.Errorf("%s: %s command cannot be used with 'at' parameter.", ctx.ProgramName, strings.Join(ctx.CommandName, " "))
		}
		return nil
	}

	if flags&subcommands.BeforeRepositoryWithStorage != 0 {
		repo, err := repository.Inexistent(ctx.App.GetInner(), ctx.StoreConfig)
		if err != nil {
			fmt.Fprintf(ctx.App.Stderr, "%s: %s\n", ctx.ProgramName, err)
			return err
		}
		manager.repository = repo
		ctx.RegisterCleanupNoErr(func() {
			if err := repo.Close(); err != nil && ctx.Logger != nil {
				ctx.Logger.Warn("could not close repository: %s", err)
			}
		})
		return nil
	}

	store, serializedConfig, err := storage.Open(ctx.App.GetInner(), ctx.StoreConfig)
	if err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s: failed to open the repository at %s: %s\n", ctx.ProgramName, ctx.StoreConfig["location"], err)
		fmt.Fprintln(ctx.App.Stderr, "To specify an alternative repository, please use \"plakar at <location> <command>\".")
		return err
	}

	repoConfig, err := storage.NewConfigurationFromWrappedBytes(serializedConfig)
	if err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s: %s\n", ctx.ProgramName, err)
		_ = store.Close(ctx.App)
		return err
	}

	if repoConfig.Version != versioning.FromString(storage.VERSION) {
		fmt.Fprintf(ctx.App.Stderr, "%s: incompatible repository version: %s != %s\n", ctx.ProgramName, repoConfig.Version, storage.VERSION)
		_ = store.Close(ctx.App)
		return fmt.Errorf("incompatible repository version")
	}

	if err := setupEncryption(ctx.App, repoConfig); err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s: %s\n", ctx.ProgramName, err)
		_ = store.Close(ctx.App)
		return err
	}

	var repo *repository.Repository
	if ctx.Options.Agentless {
		repo, err = repository.New(ctx.App.GetInner(), ctx.App.GetSecret(), store, serializedConfig)
	} else {
		repo, err = repository.NewNoRebuild(ctx.App.GetInner(), ctx.App.GetSecret(), store, serializedConfig)
	}
	if err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s: %s\n", ctx.ProgramName, err)
		_ = store.Close(ctx.App)
		return err
	}

	manager.store = store
	manager.repository = repo
	manager.openedStore = true

	ctx.RegisterCleanupNoErr(func() {
		if err := repo.Close(); err != nil && ctx.Logger != nil {
			ctx.Logger.Warn("could not close repository: %s", err)
		}
	})

	ctx.RegisterCleanupNoErr(func() {
		if err := store.Close(ctx.App); err != nil && ctx.Logger != nil {
			ctx.Logger.Warn("could not close store: %s", err)
		}
	})

	return nil
}

func getPassphraseFromEnv(ctx *appcontext.AppContext, params map[string]string) (string, error) {
	if ctx.KeyFromFile != "" {
		return ctx.KeyFromFile, nil
	}

	if pass, ok := params["passphrase"]; ok {
		delete(params, "passphrase")
		return pass, nil
	}

	if cmd, ok := params["passphrase_cmd"]; ok {
		delete(params, "passphrase_cmd")
		return utils.GetPassphraseFromCommand(cmd)
	}

	if pass, ok := os.LookupEnv("PLAKAR_PASSPHRASE"); ok {
		return pass, nil
	}

	return "", nil
}

func setupEncryption(ctx *appcontext.AppContext, config *storage.Configuration) error {
	if config.Encryption == nil {
		return nil
	}

	if ctx.KeyFromFile != "" {
		secret := []byte(ctx.KeyFromFile)
		key, err := encryption.DeriveKey(config.Encryption.KDFParams, secret)
		if err != nil {
			return err
		}

		if !encryption.VerifyCanary(config.Encryption, key) {
			return ErrCantUnlock
		}
		ctx.SetSecret(key)
		return nil
	}

	for range 3 {
		secret, err := utils.GetPassphrase("repository")
		if err != nil {
			return err
		}

		key, err := encryption.DeriveKey(config.Encryption.KDFParams, secret)
		if err != nil {
			return err
		}
		if encryption.VerifyCanary(config.Encryption, key) {
			ctx.SetSecret(key)
			return nil
		}
	}

	return ErrCantUnlock
}
