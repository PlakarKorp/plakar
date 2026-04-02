package ui

import (
	"fmt"
	"io"

	"github.com/PlakarKorp/kloset/repository"
)

var ErrUserAbort error = fmt.Errorf("aborted per user request")

type UI interface {
	SetRepository(*repository.Repository)
	Run() error
	Wait() error
	Stop()

	Stdout() io.Writer
	Stderr() io.Writer
}
