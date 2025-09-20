package bootstrap

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/denisbrodbeck/machineid"
	"github.com/google/uuid"
)

// Options contains command line options that impact bootstrap stages.
type Options struct {
	CPUCount             int
	CPUProfile           string
	MemProfile           string
	PrintExecutionTime   bool
	TraceSelectors       string
	Quiet                bool
	KeyFile              string
	Agentless            bool
	EnableSecurityCheck  bool
	DisableSecurityCheck bool
}

// ConfigContext transports state between bootstrap stages.
type ConfigContext struct {
	ProgramName string
	Args        []string

	App *appcontext.AppContext

	Options Options

	Logger *logging.Logger

	Command     subcommands.Subcommand
	CommandName []string
	CommandArgs []string

	RepositoryPath string
	StoreConfig    map[string]string

	KeyFromFile      string
	PassphraseSource string
	AtSyntax         bool
	CommandArgsRaw   []string

	ShouldExit bool
	ExitCode   int

	Profiling  *ProfilingManager
	Security   *SecurityManager
	Repository *RepositoryManager
	Signals    *SignalHandler

	closers       []func() error
	nonErrClosers []func()

	now func() time.Time
}

// NewConfigContext creates a bootstrap configuration context from argv.
func NewConfigContext(argv []string) *ConfigContext {
	program := "plakar"
	if len(argv) > 0 {
		program = filepath.Base(argv[0])
	}

	args := make([]string, 0, len(argv))
	if len(argv) > 1 {
		args = append(args, argv[1:]...)
	}

	appCtx := appcontext.NewAppContext()

	ctx := &ConfigContext{
		ProgramName: program,
		Args:        args,
		App:         appCtx,
		ExitCode:    0,
		now:         time.Now,
	}

	ctx.RegisterCleanupNoErr(func() {
		ctx.App.Close()
	})

	return ctx
}

// RegisterCleanup registers a cleanup function that may return an error.
func (c *ConfigContext) RegisterCleanup(fn func() error) {
	c.closers = append(c.closers, fn)
}

// RegisterCleanupNoErr registers a cleanup function that cannot fail.
func (c *ConfigContext) RegisterCleanupNoErr(fn func()) {
	c.nonErrClosers = append(c.nonErrClosers, fn)
}

// Close runs cleanup handlers in reverse order, aggregating errors.
func (c *ConfigContext) Close() error {
	var errs []error
	for i := len(c.closers) - 1; i >= 0; i-- {
		if err := c.closers[i](); err != nil {
			errs = append(errs, err)
		}
	}
	for i := len(c.nonErrClosers) - 1; i >= 0; i-- {
		c.nonErrClosers[i]()
	}

	return errors.Join(errs...)
}

// ConfigStage prepares the application context and CLI configuration.
type ConfigStage struct{}

// NewConfigStage returns a configuration stage.
func NewConfigStage() *ConfigStage {
	return &ConfigStage{}
}

func (s *ConfigStage) Name() string { return "configuration" }

func (s *ConfigStage) Execute(ctx *ConfigContext) error {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s\n", err)
		ctx.ExitCode = 1
		return err
	}
	ctx.App.CWD = cwd

	opCPU := runtime.GOMAXPROCS(0)
	if opCPU != 1 {
		opCPU--
	}

	currentUser, err := user.Current()
	if err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s: go away casper !\n", ctx.ProgramName)
		ctx.ExitCode = 1
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	machineID, err := machineid.ID()
	if err != nil {
		machineID = uuid.NewSHA1(uuid.Nil, []byte(hostname)).String()
	}
	machineID = strings.ToLower(machineID)

	configDir, err := utils.GetConfigDir("plakar")
	if err != nil {
		fmt.Fprintf(ctx.App.Stderr, "%s: could not get config directory: %s\n", ctx.ProgramName, err)
		ctx.ExitCode = 1
		return err
	}

	flag.CommandLine = flag.NewFlagSet(ctx.ProgramName, flag.ContinueOnError)
	flag.CommandLine.SetOutput(ctx.App.Stderr)

	var optCPUCount int
	var optConfigDir string
	var optCPUProfile string
	var optMemProfile string
	var optTime bool
	var optTrace string
	var optQuiet bool
	var optKeyFile string
	var optAgentless bool
	var optEnableSecurityCheck bool
	var optDisableSecurityCheck bool

	flag.StringVar(&optConfigDir, "config", configDir, "configuration directory")
	flag.IntVar(&optCPUCount, "cpu", opCPU, "limit the number of usable cores")
	flag.StringVar(&optCPUProfile, "profile-cpu", "", "profile CPU usage")
	flag.StringVar(&optMemProfile, "profile-mem", "", "profile MEM usage")
	flag.BoolVar(&optTime, "time", false, "display command execution time")
	flag.StringVar(&optTrace, "trace", "", "display trace logs, comma-separated (all, trace, repository, snapshot, server)")
	flag.BoolVar(&optQuiet, "quiet", false, "no output except errors")
	flag.StringVar(&optKeyFile, "keyfile", "", "use passphrase from key file when prompted")
	flag.BoolVar(&optAgentless, "no-agent", false, "run without agent")
	flag.BoolVar(&optEnableSecurityCheck, "enable-security-check", false, "enable update check")
	flag.BoolVar(&optDisableSecurityCheck, "disable-security-check", false, "disable update check")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [OPTIONS] [at REPOSITORY] COMMAND [COMMAND_OPTIONS]...\n", ctx.ProgramName)
		fmt.Fprintf(flag.CommandLine.Output(), "\nBy default, the repository is $PLAKAR_REPOSITORY or $HOME/.plakar.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nOPTIONS:\n")
		flag.PrintDefaults()

		fmt.Fprintf(flag.CommandLine.Output(), "\nCOMMANDS:\n")
		listCommands(flag.CommandLine.Output(), "  ")
		fmt.Fprintf(flag.CommandLine.Output(), "\nFor more information on a command, use '%s help COMMAND'.\n", ctx.ProgramName)
	}

	if err := flag.CommandLine.Parse(ctx.Args); err != nil {
		ctx.ExitCode = 2
		return err
	}

	app := ctx.App
	app.ConfigDir = optConfigDir
	if err := app.ReloadConfig(); err != nil {
		fmt.Fprintf(app.Stderr, "%s: could not load configuration: %s\n", ctx.ProgramName, err)
		ctx.ExitCode = 1
		return err
	}

	app.Client = "plakar/" + utils.GetVersion()

	if _, envAgentless := os.LookupEnv("PLAKAR_AGENTLESS"); envAgentless || runtime.GOOS == "windows" {
		optAgentless = true
	}

	cacheSubdir := "plakar"

	cookiesDir, err := utils.GetCacheDir(cacheSubdir)
	if err != nil {
		fmt.Fprintf(app.Stderr, "%s: could not get cookies directory: %s\n", ctx.ProgramName, err)
		ctx.ExitCode = 1
		return err
	}

	cookiesManager := cookies.NewManager(cookiesDir)
	app.SetCookies(cookiesManager)
	ctx.RegisterCleanupNoErr(func() {
		cookiesManager.Close()
	})

	if optAgentless {
		cacheSubdir = "plakar-agentless"
	}

	cacheDir, err := utils.GetCacheDir(cacheSubdir)
	if err != nil {
		fmt.Fprintf(app.Stderr, "%s: could not get cache directory: %s\n", ctx.ProgramName, err)
		ctx.ExitCode = 1
		return err
	}

	app.CacheDir = cacheDir
	cacheManager := caching.NewManager(cacheDir)
	app.SetCache(cacheManager)
	ctx.RegisterCleanupNoErr(func() {
		cacheManager.Close()
	})

	dataDir, err := utils.GetDataDir("plakar")
	if err != nil {
		fmt.Fprintf(app.Stderr, "%s: could not get data directory: %s\n", ctx.ProgramName, err)
		ctx.ExitCode = 1
		return err
	}

	app.SetPlugins(plugins.NewManager(dataDir, cookiesDir))

	if optDisableSecurityCheck {
		app.GetCookies().SetDisabledSecurityCheck()
		fmt.Fprintln(app.Stdout, "security check disabled !")
		ctx.ShouldExit = true
		ctx.ExitCode = 1
		return nil
	}

	optDisableSecurityCheck = app.GetCookies().IsDisabledSecurityCheck()

	if optEnableSecurityCheck {
		app.GetCookies().RemoveDisabledSecurityCheck()
		fmt.Fprintln(app.Stdout, "security check enabled !")
		ctx.ShouldExit = true
		ctx.ExitCode = 1
		return nil
	}

	ctx.Options = Options{
		CPUCount:             optCPUCount,
		CPUProfile:           optCPUProfile,
		MemProfile:           optMemProfile,
		PrintExecutionTime:   optTime,
		TraceSelectors:       optTrace,
		Quiet:                optQuiet,
		KeyFile:              optKeyFile,
		Agentless:            optAgentless,
		EnableSecurityCheck:  optEnableSecurityCheck,
		DisableSecurityCheck: optDisableSecurityCheck,
	}

	if optCPUCount <= 0 {
		fmt.Fprintf(app.Stderr, "%s: invalid -cpu value %d\n", ctx.ProgramName, optCPUCount)
		ctx.ExitCode = 1
		return fmt.Errorf("invalid cpu value")
	}
	if optCPUCount > runtime.NumCPU() {
		fmt.Fprintf(app.Stderr, "%s: can't use more cores than available: %d\n", ctx.ProgramName, runtime.NumCPU())
		ctx.ExitCode = 1
		return fmt.Errorf("too many cpus requested")
	}

	runtime.GOMAXPROCS(optCPUCount)

	var keyFromFile string
	if optKeyFile != "" {
		data, err := os.ReadFile(optKeyFile)
		if err != nil {
			fmt.Fprintf(app.Stderr, "%s: could not read key file: %s\n", ctx.ProgramName, err)
			ctx.ExitCode = 1
			return err
		}
		keyFromFile = strings.TrimSuffix(string(data), "\n")
	}

	app.OperatingSystem = runtime.GOOS
	app.Architecture = runtime.GOARCH
	app.Username = currentUser.Username
	app.Hostname = hostname
	app.CommandLine = strings.Join(os.Args, " ")
	app.MachineID = machineID
	app.KeyFromFile = keyFromFile
	app.ProcessID = os.Getpid()
	app.MaxConcurrency = optCPUCount*2 + 1

	if flag.CommandLine.NArg() == 0 {
		fmt.Fprintf(app.Stderr, "%s: a subcommand must be provided\n", filepath.Base(ctx.ProgramName))
		listCommands(app.Stderr, "  ")
		ctx.ExitCode = 1
		return fmt.Errorf("missing subcommand")
	}

	logger := logging.NewLogger(os.Stdout, os.Stderr)
	if !optQuiet {
		logger.EnableInfo()
	}
	if optTrace != "" {
		logger.EnableTracing(optTrace)
	}
	app.SetLogger(logger)
	ctx.Logger = logger

	if err := app.GetPlugins().LoadPlugins(app.GetInner()); err != nil {
		log.Fatalf("failed to load the plugins: %s", err)
	}

	repositoryPath, args, atSyntax, err := resolveRepositoryPath(app)
	if err != nil {
		ctx.ExitCode = 1
		return err
	}
	ctx.RepositoryPath = repositoryPath
	ctx.AtSyntax = atSyntax
	ctx.CommandArgsRaw = append([]string(nil), args...)

	cmdInput := append([]string(nil), args...)
	cmd, name, args := subcommands.Lookup(cmdInput)
	if cmd == nil {
		missing := ""
		if len(cmdInput) > 0 {
			missing = cmdInput[0]
		}
		fmt.Fprintf(app.Stderr, "command not found: %s\n", missing)
		ctx.ExitCode = 1
		return fmt.Errorf("command not found")
	}

	storeConfig, err := app.Config.GetRepository(repositoryPath)
	if err != nil {
		fmt.Fprintf(app.Stderr, "%s: %s\n", ctx.ProgramName, err)
		ctx.ExitCode = 1
		return err
	}

	passphrase, err := getPassphraseFromEnv(app, storeConfig)
	if err != nil {
		fmt.Fprintf(app.Stderr, "%s: %s\n", ctx.ProgramName, err)
		ctx.ExitCode = 1
		return err
	}
	if passphrase != "" {
		app.KeyFromFile = passphrase
	}

	ctx.Command = cmd
	ctx.CommandName = name
	ctx.CommandArgs = args
	ctx.StoreConfig = storeConfig
	ctx.KeyFromFile = app.KeyFromFile

	return nil
}

func resolveRepositoryPath(ctx *appcontext.AppContext) (string, []string, bool, error) {
	args := flag.Args()
	if len(args) == 0 {
		return "", nil, false, fmt.Errorf("no command provided")
	}

	if args[0] == "at" {
		if len(args) < 2 {
			log.Fatalf("%s: missing plakar repository", flag.CommandLine.Name())
		}
		if len(args) < 3 {
			log.Fatalf("%s: missing command", flag.CommandLine.Name())
		}
		repo := args[1]
		return repo, args[2:], true, nil
	}

	repositoryPath := os.Getenv("PLAKAR_REPOSITORY")
	if repositoryPath != "" {
		return repositoryPath, args, false, nil
	}

	if ctx.Config.DefaultRepository != "" {
		return "@" + ctx.Config.DefaultRepository, args, false, nil
	}

	userDefault, err := user.Current()
	if err != nil {
		return "", nil, false, err
	}

	return "fs:" + filepath.Join(userDefault.HomeDir, ".plakar"), args, false, nil
}

func listCommands(out interface{ Write([]byte) (int, error) }, prefix string) {
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
