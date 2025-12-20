package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/caching/pebble"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/task"
	"github.com/PlakarKorp/plakar/ui/stdio"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"

	_ "github.com/PlakarKorp/plakar/subcommands/agent"
	_ "github.com/PlakarKorp/plakar/subcommands/archive"
	_ "github.com/PlakarKorp/plakar/subcommands/backup"
	_ "github.com/PlakarKorp/plakar/subcommands/cat"
	_ "github.com/PlakarKorp/plakar/subcommands/check"
	_ "github.com/PlakarKorp/plakar/subcommands/clone"
	_ "github.com/PlakarKorp/plakar/subcommands/config"
	_ "github.com/PlakarKorp/plakar/subcommands/create"
	_ "github.com/PlakarKorp/plakar/subcommands/diag"
	_ "github.com/PlakarKorp/plakar/subcommands/diff"
	_ "github.com/PlakarKorp/plakar/subcommands/digest"
	_ "github.com/PlakarKorp/plakar/subcommands/dup"
	_ "github.com/PlakarKorp/plakar/subcommands/help"
	_ "github.com/PlakarKorp/plakar/subcommands/info"
	_ "github.com/PlakarKorp/plakar/subcommands/locate"
	_ "github.com/PlakarKorp/plakar/subcommands/login"
	_ "github.com/PlakarKorp/plakar/subcommands/ls"
	_ "github.com/PlakarKorp/plakar/subcommands/maintenance"
	_ "github.com/PlakarKorp/plakar/subcommands/mount"
	_ "github.com/PlakarKorp/plakar/subcommands/pkg"
	_ "github.com/PlakarKorp/plakar/subcommands/prune"
	_ "github.com/PlakarKorp/plakar/subcommands/ptar"
	_ "github.com/PlakarKorp/plakar/subcommands/repair"
	_ "github.com/PlakarKorp/plakar/subcommands/restore"
	_ "github.com/PlakarKorp/plakar/subcommands/rm"
	_ "github.com/PlakarKorp/plakar/subcommands/scheduler"
	_ "github.com/PlakarKorp/plakar/subcommands/server"
	_ "github.com/PlakarKorp/plakar/subcommands/service"
	_ "github.com/PlakarKorp/plakar/subcommands/ui"
	_ "github.com/PlakarKorp/plakar/subcommands/version"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	_ "github.com/PlakarKorp/integration-fs/importer"
	_ "github.com/PlakarKorp/integration-fs/storage"
	_ "github.com/PlakarKorp/integration-ptar/storage"
	_ "github.com/PlakarKorp/integration-stdio/exporter"
	_ "github.com/PlakarKorp/integration-stdio/importer"
	_ "github.com/PlakarKorp/integration-tar/importer"
)

var ErrCantUnlock = errors.New("failed to unlock repository")

func entryPoint() int {
	// default values
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	opt_cpuDefault := runtime.GOMAXPROCS(0)
	if opt_cpuDefault != 1 {
		opt_cpuDefault = opt_cpuDefault - 1
	}

	userDefault, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: go away casper !\n", flag.CommandLine.Name())
		return 1
	}

	hostnameDefault, err := os.Hostname()
	if err != nil {
		hostnameDefault = "localhost"
	}

	opt_machineIdDefault, err := machineid.ID()
	if err != nil {
		opt_machineIdDefault = uuid.NewSHA1(uuid.Nil, []byte(hostnameDefault)).String()
	}
	opt_machineIdDefault = strings.ToLower(opt_machineIdDefault)

	opt_configDefault, err := utils.GetConfigDir("plakar")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not get config directory: %s\n", flag.CommandLine.Name(), err)
		return 1
	}

	// command line overrides
	var opt_cpuCount int
	var opt_configdir string
	var opt_cpuProfile string
	var opt_memProfile string
	var opt_time bool
	var opt_trace string
	var opt_quiet bool
	var opt_keyfile string
	var opt_enableSecurityCheck bool
	var opt_disableSecurityCheck bool
	var opt_maxConcurrency int

	flag.StringVar(&opt_configdir, "config", opt_configDefault, "configuration directory")
	flag.IntVar(&opt_cpuCount, "cpu", opt_cpuDefault, "limit the number of usable cores")
	flag.IntVar(&opt_maxConcurrency, "concurrency", -1, "limit the number of concurrent operations")
	flag.StringVar(&opt_cpuProfile, "profile-cpu", "", "profile CPU usage")
	flag.StringVar(&opt_memProfile, "profile-mem", "", "profile MEM usage")
	flag.BoolVar(&opt_time, "time", false, "display command execution time")
	flag.StringVar(&opt_trace, "trace", "", "display trace logs, comma-separated (all, trace, repository, snapshot, server)")
	flag.BoolVar(&opt_quiet, "quiet", false, "no output except errors")
	flag.StringVar(&opt_keyfile, "keyfile", "", "use passphrase from key file when prompted")
	flag.BoolVar(&opt_enableSecurityCheck, "enable-security-check", false, "enable update check")
	flag.BoolVar(&opt_disableSecurityCheck, "disable-security-check", false, "disable update check")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTIONS] [at REPOSITORY] COMMAND [COMMAND_OPTIONS]...\n", flag.CommandLine.Name())
		fmt.Fprintf(flag.CommandLine.Output(), "\nBy default, the repository is $PLAKAR_REPOSITORY or $HOME/.plakar.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nOPTIONS:\n")
		flag.PrintDefaults()

		fmt.Fprintf(flag.CommandLine.Output(), "\nCOMMANDS:\n")
		listCmds(flag.CommandLine.Output(), "  ")
		fmt.Fprintf(flag.CommandLine.Output(), "\nFor more information on a command, use '%s help COMMAND'.\n", flag.CommandLine.Name())
	}
	flag.Parse()

	ctx := appcontext.NewAppContext()
	defer stdio.Run(ctx)()
	defer ctx.Close()

	ctx.ConfigDir = opt_configdir
	err = ctx.ReloadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not load configuration: %s\n", flag.CommandLine.Name(), err)
		return 1
	}

	ctx.Client = "plakar/" + utils.GetVersion()
	ctx.CWD = cwd

	// default cachedir
	cacheSubDir := "plakar"

	cookiesDir, err := utils.GetCacheDir(cacheSubDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not get cookies directory: %s\n", flag.CommandLine.Name(), err)
		return 1
	}

	ctx.SetCookies(cookies.NewManager(cookiesDir))
	defer ctx.GetCookies().Close()

	cacheDir, err := utils.GetCacheDir(cacheSubDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not get cache directory: %s\n", flag.CommandLine.Name(), err)
		return 1
	}
	ctx.CacheDir = cacheDir
	ctx.SetCache(caching.NewManager(pebble.Constructor(cacheDir)))
	defer ctx.GetCache().Close()

	dataDir, err := utils.GetDataDir("plakar")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not get data directory: %s\n", flag.CommandLine.Name(), err)
		return 1
	}
	ctx.SetPlugins(plugins.NewManager(dataDir, cookiesDir))

	if opt_disableSecurityCheck {
		ctx.GetCookies().SetDisabledSecurityCheck()
		fmt.Fprintln(ctx.Stdout, "security check disabled !")
		return 1
	} else {
		opt_disableSecurityCheck = ctx.GetCookies().IsDisabledSecurityCheck()
	}

	if opt_enableSecurityCheck {
		ctx.GetCookies().RemoveDisabledSecurityCheck()
		fmt.Fprintln(ctx.Stdout, "security check enabled !")
		return 1
	}

	checkUpdate(ctx, opt_disableSecurityCheck)

	// setup from default + override
	if opt_cpuCount <= 0 {
		fmt.Fprintf(os.Stderr, "%s: invalid -cpu value %d\n", flag.CommandLine.Name(), opt_cpuCount)
		return 1
	}
	if opt_cpuCount > runtime.NumCPU() {
		fmt.Fprintf(os.Stderr, "%s: can't use more cores than available: %d\n", flag.CommandLine.Name(), runtime.NumCPU())
		return 1
	}
	runtime.GOMAXPROCS(opt_cpuCount)

	if opt_maxConcurrency == 0 {
		fmt.Fprintf(os.Stderr, "%s: invalid -concurrency value %d\n", flag.CommandLine.Name(), opt_maxConcurrency)
		return 1
	}
	if opt_maxConcurrency == -1 {
		opt_maxConcurrency = opt_cpuCount
	}

	if opt_cpuProfile != "" {
		f, err := os.Create(opt_cpuProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: could not create CPU profile: %s\n", flag.CommandLine.Name(), err)
			return 1
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "%s: could not start CPU profile: %s\n", flag.CommandLine.Name(), err)
			return 1
		}
		defer pprof.StopCPUProfile()
	}

	var secretFromKeyfile string
	if opt_keyfile != "" {
		data, err := os.ReadFile(opt_keyfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: could not read key file: %s\n", flag.CommandLine.Name(), err)
			return 1
		}
		secretFromKeyfile = strings.TrimSuffix(string(data), "\n")
	}

	ctx.OperatingSystem = runtime.GOOS
	ctx.Architecture = runtime.GOARCH
	ctx.Username = userDefault.Username
	ctx.Hostname = hostnameDefault
	ctx.CommandLine = strings.Join(os.Args, " ")
	ctx.MachineID = opt_machineIdDefault
	ctx.KeyFromFile = secretFromKeyfile
	ctx.ProcessID = os.Getpid()
	ctx.MaxConcurrency = opt_maxConcurrency

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: a subcommand must be provided\n", filepath.Base(flag.CommandLine.Name()))
		listCmds(os.Stderr, "  ")
		return 1
	}

	logger := logging.NewLogger(os.Stdout, os.Stderr)

	// start logging
	if !opt_quiet {
		logger.EnableInfo()
	}
	if opt_trace != "" {
		logger.EnableTracing(opt_trace)
	}

	ctx.SetLogger(logger)

	err = ctx.GetPlugins().LoadPlugins(ctx.GetInner())
	if err != nil {
		log.Fatalf("failed to load the plugins: %s", err)
	}

	var repositoryPath string

	var at bool
	var args []string
	if flag.Arg(0) == "at" {
		if len(flag.Args()) < 2 {
			log.Fatalf("%s: missing plakar repository", flag.CommandLine.Name())
		}
		if len(flag.Args()) < 3 {
			log.Fatalf("%s: missing command", flag.CommandLine.Name())
		}
		repositoryPath = flag.Arg(1)
		args = flag.Args()[2:]
		at = true
	} else {
		repositoryPath = os.Getenv("PLAKAR_REPOSITORY")
		if repositoryPath == "" {
			def := ctx.Config.DefaultRepository
			if def != "" {
				repositoryPath = "@" + def
			} else {
				repositoryPath = "fs:" + filepath.Join(userDefault.HomeDir, ".plakar")
			}
		}

		args = flag.Args()
	}

	storeConfig, err := ctx.Config.GetRepository(repositoryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return 1
	}

	cmd, name, args := subcommands.Lookup(args)
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "command not found: %s\n", args[0])
		return 1
	}

	// try to get the passphrase from env and store config so that it's
	// available to subcommands like create.
	passphrase, err := getPassphraseFromEnv(ctx, storeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return 1
	}
	if passphrase != "" {
		ctx.KeyFromFile = passphrase
	}

	var store storage.Store
	var repo *repository.Repository

	if cmd.GetFlags()&subcommands.BeforeRepositoryOpen != 0 {
		if at {
			log.Fatalf("%s: %s command cannot be used with 'at' parameter.",
				flag.CommandLine.Name(), strings.Join(name, " "))
		}
		// store and repo can stay nil
	} else if cmd.GetFlags()&subcommands.BeforeRepositoryWithStorage != 0 {
		repo, err = repository.Inexistent(ctx.GetInner(), storeConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
			return 1
		}
	} else {
		var serializedConfig []byte
		store, serializedConfig, err = storage.Open(ctx.GetInner(), storeConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to open the repository at %s: %s\n", flag.CommandLine.Name(), storeConfig["location"], err)
			fmt.Fprintln(os.Stderr, "To specify an alternative repository, please use \"plakar at <location> <command>\".")
			return 1
		}

		repoConfig, err := storage.NewConfigurationFromWrappedBytes(serializedConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
			return 1
		}

		if repoConfig.Version != versioning.FromString(storage.VERSION) {
			fmt.Fprintf(os.Stderr, "%s: incompatible repository version: %s != %s\n",
				flag.CommandLine.Name(), repoConfig.Version, storage.VERSION)
			return 1
		}

		if err := setupEncryption(ctx, repoConfig); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
			return 1
		}

		// Actual rebuild is done by cached.
		repo, err = repository.NewNoRebuild(ctx.GetInner(), ctx.GetSecret(), store, serializedConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
			return 1
		}
	}

	ctx.StoreConfig = storeConfig

	t0 := time.Now()
	if err := cmd.Parse(ctx, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return 1
	}

	cmd.SetCWD(ctx.CWD)
	cmd.SetCommandLine(ctx.CommandLine)

	c := make(chan os.Signal, 1)
	go func() {
		<-c
		fmt.Fprintf(os.Stderr, "%s: Interrupting, it might take a while...\n", flag.CommandLine.Name())
		ctx.Cancel(fmt.Errorf("interrupted"))
	}()
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var status int

	// If we are working on a repo, rebuild the state.
	if cmd.GetFlags()&subcommands.BeforeRepositoryOpen == 0 && cmd.GetFlags()&subcommands.BeforeRepositoryWithStorage == 0 {
		_, err = agent.RebuildStateFromCached(ctx, repo.Configuration().RepositoryID, storeConfig)
		if err == nil {
			status, err = task.RunCommand(ctx, cmd, repo, "@agentless")
		}
	} else {
		status, err = task.RunCommand(ctx, cmd, repo, "@agentless")
	}

	t1 := time.Since(t0)

	if err != nil {
		if errors.Is(err, context.Canceled) && ctx.Err() != nil {
			err = context.Cause(ctx)
		}

		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), utils.SanitizeText(err.Error()))
		if errors.Is(err, agent.ErrWrongVersion) {
			fmt.Fprintln(os.Stderr, "To stop the current agent, run:")
			fmt.Fprintln(os.Stderr, "\t$ plakar agent stop")
		}
	}

	if repo != nil {
		err = repo.Close()
		if err != nil {
			logger.Warn("could not close repository: %s", err)
		}
	}

	if store != nil {
		err = store.Close(ctx)
		if err != nil {
			logger.Warn("could not close store: %s", err)
		}
	}

	if opt_time {
		fmt.Println("time:", t1)
	}

	if opt_memProfile != "" {
		f, err := os.Create(opt_memProfile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "%s: could not write MEM profile: %d\n", flag.CommandLine.Name(), err)
			return 1
		}
	}

	return status
}

func checkUpdate(ctx *appcontext.AppContext, disableSecurityCheck bool) {
	if ctx.GetCookies().IsFirstRun() {
		ctx.GetCookies().SetFirstRun()
		if disableSecurityCheck {
			return
		}

		fmt.Fprintln(ctx.Stdout, "Welcome to plakar !")
		fmt.Fprintln(ctx.Stdout, "")
		fmt.Fprintln(ctx.Stdout, "By default, plakar checks for security updates on the releases feed once every 24h.")
		fmt.Fprintln(ctx.Stdout, "It will notify you if there are important updates that you need to install.")
		fmt.Fprintln(ctx.Stdout, "")
		fmt.Fprintln(ctx.Stdout, "If you prefer to watch yourself, you can disable this permanently by running:")
		fmt.Fprintln(ctx.Stdout, "")
		fmt.Fprintln(ctx.Stdout, "\tplakar -disable-security-check")
		fmt.Fprintln(ctx.Stdout, "")
		fmt.Fprintln(ctx.Stdout, "If you change your mind, run:")
		fmt.Fprintln(ctx.Stdout, "")
		fmt.Fprintln(ctx.Stdout, "\tplakar -enable-security-check")
		fmt.Fprintln(ctx.Stdout, "")
		fmt.Fprintln(ctx.Stdout, "EOT")
		return
	}

	if disableSecurityCheck {
		return
	}

	// best effort check if security or reliability fix have been issued
	rus, err := utils.CheckUpdate(ctx.CacheDir)
	if err != nil {
		return
	}
	if !rus.SecurityFix && !rus.ReliabilityFix {
		return
	}

	concerns := ""
	if rus.SecurityFix {
		concerns = "security"
	}
	if rus.ReliabilityFix {
		if concerns != "" {
			concerns += " and "
		}
		concerns += "reliability"
	}
	fmt.Fprintf(os.Stderr, "WARNING: %s concerns affect your current version, please upgrade to %s (+%d releases).\n",
		concerns, rus.Latest, rus.FoundCount)
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
		key, err := encryption.DeriveKey(config.Encryption.KDFParams,
			secret)
		if err != nil {
			return err
		}

		if !encryption.VerifyCanary(config.Encryption, key) {
			return ErrCantUnlock
		}
		ctx.SetSecret(key)
		return nil
	}

	// fall back to prompting
	for range 3 {
		secret, err := utils.GetPassphrase("repository")
		if err != nil {
			return err
		}

		key, err := encryption.DeriveKey(config.Encryption.KDFParams,
			secret)
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

func listCmds(out io.Writer, prefix string) {
	var last string
	var subs []string

	flush := func() {
		pre, post := " ", ""
		if len(subs) > 1 && subs[0] == "" {
			pre, post = " [", "]"
			subs = subs[1:]
		}
		subcmds := strings.Join(subs, " | ")
		fmt.Fprint(out, prefix, last, pre, subcmds, post, "\n")
	}

	all := subcommands.List()
	for _, cmd := range all {
		if len(cmd) == 0 || cmd[0] == "diag" {
			continue
		}

		if last == "" {
			goto next
		}

		if last == cmd[0] {
			if len(subs) > 0 && subs[len(subs)-1] != cmd[1] {
				subs = append(subs, cmd[1])
			}
			continue
		}

		flush()

	next:
		subs = subs[:0]
		last = cmd[0]
		if len(cmd) > 1 {
			subs = append(subs, cmd[1])
		} else {
			subs = append(subs, "")
		}
	}
	flush()
}

func main() {
	os.Exit(entryPoint())
}
