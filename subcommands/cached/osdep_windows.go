package cached

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/PlakarKorp/plakar/appcontext"
)

// setupSyslog is a no-op on Windows. cached runs detached (no console), and
// writes to a logfile when -log is given; without it logs go to io.Discard,
// same as on Unix when syslog setup fails.
func setupSyslog(ctx *appcontext.AppContext) error {
	return nil
}

// Windows process creation flags. See:
// https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags
const (
	detachedProcess       = 0x00000008
	createNewProcessGroup = 0x00000200
	createNoWindow        = 0x08000000
)

// daemonize re-execs the current binary detached from the calling console,
// then exits. The child runs without an attached console and outlives the
// shell that started it — the same lifecycle the Unix fork+setsid path
// provides.
func daemonize(argv []string) error {
	binary, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(binary, argv[1:]...)
	cmd.Env = append(os.Environ(), "REEXEC=1")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: detachedProcess | createNewProcessGroup | createNoWindow,
	}
	// A detached process has no console, so no inherited stdio.
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil

	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
