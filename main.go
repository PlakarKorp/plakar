package main

import (
	"flag"
	"fmt"
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

	"github.com/PlakarKorp/kloset/appcontext"
	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/config"
	"github.com/PlakarKorp/kloset/cookies"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/PlakarKorp/plakar/agent"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/task"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"

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
	_ "github.com/PlakarKorp/plakar/subcommands/help"
	_ "github.com/PlakarKorp/plakar/subcommands/info"
	_ "github.com/PlakarKorp/plakar/subcommands/locate"
	_ "github.com/PlakarKorp/plakar/subcommands/login"
	_ "github.com/PlakarKorp/plakar/subcommands/ls"
	_ "github.com/PlakarKorp/plakar/subcommands/maintenance"
	_ "github.com/PlakarKorp/plakar/subcommands/mount"
	_ "github.com/PlakarKorp/plakar/subcommands/ptar"
	_ "github.com/PlakarKorp/plakar/subcommands/restore"
	_ "github.com/PlakarKorp/plakar/subcommands/rm"
	_ "github.com/PlakarKorp/plakar/subcommands/server"
	_ "github.com/PlakarKorp/plakar/subcommands/services"
	_ "github.com/PlakarKorp/plakar/subcommands/ui"
	_ "github.com/PlakarKorp/plakar/subcommands/version"

	_ "github.com/PlakarKorp/plakar/connectors/fs"
	_ "github.com/PlakarKorp/plakar/connectors/ftp"
	_ "github.com/PlakarKorp/plakar/connectors/http"
	_ "github.com/PlakarKorp/plakar/connectors/ptar"
	_ "github.com/PlakarKorp/plakar/connectors/s3"
	_ "github.com/PlakarKorp/plakar/connectors/sftp"
	_ "github.com/PlakarKorp/plakar/connectors/sqlite"
	_ "github.com/PlakarKorp/plakar/connectors/stdio"
)

func EntryPoint() int {
	// default values
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}
	cwd, err = utils.NormalizePath(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	opt_cpuDefault := runtime.GOMAXPROCS(0)
	if opt_cpuDefault != 1 {
		opt_cpuDefault = opt_cpuDefault - 1
	}

	opt_userDefault, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: go away casper !\n", flag.CommandLine.Name())
		return 1
	}

	opt_hostnameDefault, err := os.Hostname()
	if err != nil {
		opt_hostnameDefault = "localhost"
	}

	opt_machineIdDefault, err := machineid.ID()
	if err != nil {
		opt_machineIdDefault = uuid.NewSHA1(uuid.Nil, []byte(opt_hostnameDefault)).String()
	}
	opt_machineIdDefault = strings.ToLower(opt_machineIdDefault)

	opt_usernameDefault := opt_userDefault.Username

	configDir, err := utils.GetConfigDir("plakar")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not get config directory: %s\n", flag.CommandLine.Name(), err)
		return 1
	}
	opt_configDefault := filepath.Join(configDir, "plakar.yml")

	// command line overrides
	var opt_cpuCount int
	var opt_configfile string
	var opt_username string
	var opt_hostname string
	var opt_cpuProfile string
	var opt_memProfile string
	var opt_time bool
	var opt_trace string
	var opt_quiet bool
	var opt_keyfile string
	var opt_agentless bool
	var opt_enableSecurityCheck bool
	var opt_disableSecurityCheck bool

	flag.StringVar(&opt_configfile, "config", opt_configDefault, "configuration file")
	flag.IntVar(&opt_cpuCount, "cpu", opt_cpuDefault, "limit the number of usable cores")
	flag.StringVar(&opt_username, "username", opt_usernameDefault, "default username")
	flag.StringVar(&opt_hostname, "hostname", opt_hostnameDefault, "default hostname")
	flag.StringVar(&opt_cpuProfile, "profile-cpu", "", "profile CPU usage")
	flag.StringVar(&opt_memProfile, "profile-mem", "", "profile MEM usage")
	flag.BoolVar(&opt_time, "time", false, "display command execution time")
	flag.StringVar(&opt_trace, "trace", "", "display trace logs, comma-separated (all, trace, repository, snapshot, server)")
	flag.BoolVar(&opt_quiet, "quiet", false, "no output except errors")
	flag.StringVar(&opt_keyfile, "keyfile", "", "use passphrase from key file when prompted")
	flag.BoolVar(&opt_agentless, "no-agent", false, "run without agent")
	flag.BoolVar(&opt_enableSecurityCheck, "enable-security-check", false, "enable update check")
	flag.BoolVar(&opt_disableSecurityCheck, "disable-security-check", false, "disable update check")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTIONS] [at REPOSITORY] COMMAND [COMMAND_OPTIONS]...\n", flag.CommandLine.Name())
		fmt.Fprintf(flag.CommandLine.Output(), "\nBy default, the repository is $PLAKAR_REPOSITORY or $HOME/.plakar.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nOPTIONS:\n")
		flag.PrintDefaults()

		fmt.Fprintf(flag.CommandLine.Output(), "\nCOMMANDS:\n")
		for _, k := range subcommands.List() {
			fmt.Fprintf(flag.CommandLine.Output(), "  %s\n", k)
		}
		fmt.Fprintf(flag.CommandLine.Output(), "\nFor more information on a command, use '%s help COMMAND'.\n", flag.CommandLine.Name())
	}
	flag.Parse()

	ctx := appcontext.NewAppContext()
	defer ctx.Close()

	cfg, err := config.LoadOrCreate(opt_configfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not load configuration: %s\n", flag.CommandLine.Name(), err)
		return 1
	}

	ctx.SetConfig(cfg)

	ctx.Client = "plakar/" + utils.GetVersion()
	ctx.CWD = cwd
	ctx.KeyringDir = filepath.Join(opt_userDefault.HomeDir, ".plakar-keyring")

	_, envAgentLess := os.LookupEnv("PLAKAR_AGENTLESS")
	if envAgentLess {
		opt_agentless = true
	}

	// default cachedir
	cacheSubDir := "plakar"

	cookiesDir, err := utils.GetCacheDir(cacheSubDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not get cookies directory: %s\n", flag.CommandLine.Name(), err)
		return 1
	}
	ctx.CookiesDir = cookiesDir
	ctx.SetCookies(cookies.NewManager(cookiesDir))
	defer ctx.GetCookies().Close()

	if opt_agentless {
		cacheSubDir = "plakar-agentless"
	}
	cacheDir, err := utils.GetCacheDir(cacheSubDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not get cache directory: %s\n", flag.CommandLine.Name(), err)
		return 1
	}
	ctx.CacheDir = cacheDir
	ctx.SetCache(caching.NewManager(cacheDir))
	defer ctx.GetCache().Close()

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

	if firstRun := ctx.GetCookies().IsFirstRun(); firstRun {
		ctx.GetCookies().SetFirstRun()
		if !opt_disableSecurityCheck {
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
		}
	}

	// best effort check if security or reliability fix have been issued
	if opt_disableSecurityCheck {
		if rus, err := utils.CheckUpdate(ctx.CacheDir); err == nil {
			if rus.SecurityFix || rus.ReliabilityFix {
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
				fmt.Fprintf(os.Stderr, "WARNING: %s concerns affect your current version, please upgrade to %s (+%d releases).\n", concerns, rus.Latest, rus.FoundCount)
			}
		}
	}

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
	ctx.NumCPU = opt_cpuCount
	ctx.Username = opt_username
	ctx.Hostname = opt_hostname
	ctx.CommandLine = strings.Join(os.Args, " ")
	ctx.MachineID = opt_machineIdDefault
	ctx.KeyFromFile = secretFromKeyfile
	ctx.HomeDir = opt_userDefault.HomeDir
	ctx.ProcessID = os.Getpid()
	ctx.MaxConcurrency = ctx.NumCPU*8 + 1

	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: a subcommand must be provided\n", filepath.Base(flag.CommandLine.Name()))
		for _, k := range subcommands.List() {
			fmt.Fprintf(os.Stderr, "  %s\n", k)
		}

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
			def := ctx.GetDefaultRepositoryName()
			if def != "" {
				repositoryPath = "@" + def
			} else {
				repositoryPath = filepath.Join(ctx.HomeDir, ".plakar")
			}
		}

		args = flag.Args()
	}

	storeConfig, err := ctx.GetRepositoryConfig(repositoryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return 1
	}

	cmd, name, args := subcommands.Lookup(args)
	if cmd == nil {
		fmt.Fprintf(os.Stderr, "command not found: %s\n", args[0])
		return 1
	}

	// these commands need to be ran before the repository is opened

	var store storage.Store
	var repo *repository.Repository

	if cmd.GetFlags()&subcommands.BeforeRepositoryOpen != 0 {
		if at {
			log.Fatalf("%s: %s command cannot be used with 'at' parameter.",
				flag.CommandLine.Name(), strings.Join(name, " "))
		}
		// store and repo can stay nil
	} else if cmd.GetFlags()&subcommands.BeforeRepositoryWithStorage != 0 {
		repo, err = repository.Inexistent(ctx, storeConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
			return 1
		}
	} else {
		var serializedConfig []byte
		store, serializedConfig, err = storage.Open(ctx, storeConfig)
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

		setupEncryption(ctx, repoConfig, storeConfig)

		if opt_agentless {
			repo, err = repository.New(ctx, store, serializedConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
				return 1
			}
		} else {
			repo, err = repository.NewNoRebuild(ctx, store, serializedConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
				return 1
			}
		}
	}

	t0 := time.Now()
	if err := cmd.Parse(ctx, args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return 1
	}

	c := make(chan os.Signal, 1)
	go func() {
		<-c
		fmt.Fprintf(os.Stderr, "%s: Interrupting, it might take a while...\n", flag.CommandLine.Name())
		ctx.Cancel()
	}()
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var status int

	runWithoutAgent := opt_agentless || cmd.GetFlags()&subcommands.AgentSupport == 0
	if runWithoutAgent {
		status, err = task.RunCommand(ctx, cmd, repo, "@agentless")
	} else {
		status, err = agent.ExecuteRPC(ctx, name, cmd, storeConfig)
	}

	t1 := time.Since(t0)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), utils.SanitizeText(err.Error()))
	}

	if repo != nil {
		err = repo.Close()
		if err != nil {
			logger.Warn("could not close repository: %s", err)
		}
	}

	if store != nil {
		err = store.Close()
		if err != nil {
			logger.Warn("could not close repository: %s", err)
		}
	}

	ctx.Close()

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

func setupEncryption(ctx *appcontext.AppContext, repoConfig *storage.Configuration, storeConfig map[string]string) {

	if repoConfig.Encryption == nil {
		return
	}

	var err error
	var secret []byte

	derived := false
	envPassphrase := os.Getenv("PLAKAR_PASSPHRASE")
	if ctx.KeyFromFile == "" {
		if passphrase, ok := storeConfig["passphrase"]; ok {
			key, err := encryption.DeriveKey(repoConfig.Encryption.KDFParams, []byte(passphrase))
			if err == nil {
				if encryption.VerifyCanary(repoConfig.Encryption, key) {
					secret = key
					derived = true
				}
			}
		} else {
			for attempts := 0; attempts < 3; attempts++ {
				var passphrase []byte
				if envPassphrase == "" {
					passphrase, err = utils.GetPassphrase("repository")
					if err != nil {
						break
					}
				} else {
					passphrase = []byte(envPassphrase)
				}

				key, err := encryption.DeriveKey(repoConfig.Encryption.KDFParams, passphrase)
				if err != nil {
					continue
				}
				if !encryption.VerifyCanary(repoConfig.Encryption, key) {
					if envPassphrase != "" {
						break
					}
					continue
				}
				secret = key
				derived = true
				break
			}
		}
	} else {
		key, err := encryption.DeriveKey(repoConfig.Encryption.KDFParams, []byte(ctx.KeyFromFile))
		if err == nil {
			if encryption.VerifyCanary(repoConfig.Encryption, key) {
				secret = key
				derived = true
			}
		}
	}

	if !derived {
		fmt.Fprintf(os.Stderr, "%s: could not derive secret\n", flag.CommandLine.Name())
		os.Exit(1)
	}

	ctx.SetSecret(secret)
}

func main() {
	os.Exit(EntryPoint())
}
