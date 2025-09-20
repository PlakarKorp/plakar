package bootstrap

import (
	"fmt"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
)

// SecurityManager drives security-related bootstrap tasks.
type SecurityManager struct {
	disableChecks bool
}

// NewSecurityManager creates a new security manager.
func NewSecurityManager(disable bool) *SecurityManager {
	return &SecurityManager{disableChecks: disable}
}

// Run performs security checks according to the current configuration.
func (m *SecurityManager) Run(ctx *ConfigContext) {
	checkUpdate(ctx.App, m.disableChecks)
}

// SecurityStage wraps the security manager in a pipeline stage.
type SecurityStage struct{}

// NewSecurityStage constructs a security stage.
func NewSecurityStage() *SecurityStage {
	return &SecurityStage{}
}

func (s *SecurityStage) Name() string { return "security" }

func (s *SecurityStage) Execute(ctx *ConfigContext) error {
	manager := NewSecurityManager(ctx.Options.DisableSecurityCheck)
	ctx.Security = manager
	manager.Run(ctx)
	return nil
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
	fmt.Fprintf(ctx.Stderr, "WARNING: %s concerns affect your current version, please upgrade to %s (+%d releases).\n",
		concerns, rus.Latest, rus.FoundCount)
}
