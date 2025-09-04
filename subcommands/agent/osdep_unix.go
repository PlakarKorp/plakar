//go:build !windows

package agent

import (
	"log/syslog"
	"os"
	"syscall"

	"github.com/PlakarKorp/plakar/appcontext"
)

func setupSyslog(ctx *appcontext.AppContext) error {
	w, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, "plakar")
	if err != nil {
		return err
	}
	ctx.GetLogger().SetSyslogOutput(w)
	return nil
}

func daemonize(argv []string) error {
	binary, err := os.Executable()
	if err != nil {
		return err
	}

	procAttr := syscall.ProcAttr{
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	}
	procAttr.Files = []uintptr{
		uintptr(syscall.Stdin),
		uintptr(syscall.Stdout),
		uintptr(syscall.Stderr),
	}
	procAttr.Env = append(os.Environ(),
		"REEXEC=1",
	)

	if _, err := syscall.ForkExec(binary, argv, &procAttr); err != nil {
		return err
	} else {
		os.Exit(0)
		return nil
	}
}

func kill_self() error {
	return syscall.Kill(os.Getpid(), syscall.SIGINT)
}
