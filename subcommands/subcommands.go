package subcommands

import (
	"fmt"
	"slices"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
)

type CommandFlags uint32

const (
	NeedRepositoryKey CommandFlags = 1 << iota
	BeforeRepositoryWithStorage
	BeforeRepositoryOpen
)

type ErrCode struct {
	Err  error
	Code int
}

func NewErrCode(code int, reason string, args ...any) *ErrCode {
	return &ErrCode{Err: fmt.Errorf(reason, args...), Code: code}
}

func (e *ErrCode) Error() string  { return e.Err.Error() }
func (e *ErrCode) String() string { return e.Error() }

type Subcommand func(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error

type subcmd struct {
	args  []string
	nargs int
	flags CommandFlags
	fn    Subcommand
}

var subcommands []subcmd = make([]subcmd, 0)

func Register(fn Subcommand, flags CommandFlags, args ...string) {
	if len(args) == 0 {
		panic("can't register commands with zero arguments")
	}

	subcommands = append(subcommands, subcmd{
		args:  args,
		nargs: len(args),
		flags: flags,
		fn:    fn,
	})
}

func Lookup(arguments []string) (Subcommand, CommandFlags, []string, []string) {
	nargs := len(arguments)
	for _, subcmd := range subcommands {
		if nargs < subcmd.nargs {
			continue
		}

		if !slices.Equal(subcmd.args, arguments[:subcmd.nargs]) {
			continue
		}

		return subcmd.fn, subcmd.flags, arguments[:subcmd.nargs], arguments[subcmd.nargs:]
	}

	return nil, 0, nil, arguments
}

func List() [][]string {
	var list [][]string
	slices.SortFunc(subcommands, func(a, b subcmd) int {
		var i int
		for {
			n := strings.Compare(a.args[i], b.args[i])
			if n != 0 {
				return n
			}

			i++
			if i == len(a.args) {
				return -1
			}
			if i == len(b.args) {
				return +1
			}
		}
	})
	for _, command := range subcommands {
		list = append(list, command.args)
	}
	return list
}
