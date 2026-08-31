package subcommands

import (
	goflag "flag"
	"fmt"
	"slices"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type CommandFlags uint32

const (
	NeedRepositoryKey CommandFlags = 1 << iota
	BeforeRepositoryWithStorage
	BeforeRepositoryOpen
)

// Subcommand is the contract every plakar command implements.  Parse must not
// stash anything that only makes sense in this process -- a *cobra.Command, a
// flag set, an open repository -- so that a parsed command stays a plain
// description of the work and can be executed elsewhere.
type Subcommand interface {
	CobraCommand() *cobra.Command

	Parse(ctx *appcontext.AppContext, args []string) error
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
	GetRepositorySecret() []byte
	GetFlags() CommandFlags
	setFlags(CommandFlags)
}

type SubcommandBase struct {
	RepositorySecret []byte
	Flags            CommandFlags
}

func (cmd *SubcommandBase) setFlags(flags CommandFlags) {
	cmd.Flags = flags
}

func (cmd *SubcommandBase) GetFlags() CommandFlags {
	return cmd.Flags
}

func (cmd *SubcommandBase) GetRepositorySecret() []byte {
	return cmd.RepositorySecret
}

// SingleDash rewrites our single-dash options into the double-dash form pflag
// expects, which it would otherwise read as bundles of shorthands.  Length is
// not the criterion: -o, -k and -u are declared with StringVar and friends, so
// they are one-letter long options rather than shorthands.
//
func SingleDash(flags *pflag.FlagSet, args []string) ([]string, error) {
	out := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			return append(out, args[i:]...), nil
		}

		if !strings.HasPrefix(arg, "-") {
			return append(out, args[i:]...), nil
		}

		if len(arg) > 1 && arg[1] != '-' {
			name, _, explicit := strings.Cut(arg[1:], "=")
			f := flags.Lookup(name)
			if f == nil {
				// A real shorthand is pflag's to resolve.
				if len(name) == 1 && flags.ShorthandLookup(name) != nil {
					out = append(out, arg)
					continue
				}
				return nil, fmt.Errorf("flag provided but not defined: -%s", name)
			}
			out = append(out, "-"+arg)

			// "-name value" takes the next argument with it.
			if !explicit && f.Value.Type() != "bool" && i+1 < len(args) {
				out = append(out, args[i+1])
				i++
			}
			continue
		}

		out = append(out, arg)
	}

	return out, nil
}

// InstallGoFlags adopts the options kloset's LocateOptions installs into a
// flag.FlagSet.
func InstallGoFlags(flags *pflag.FlagSet, install func(*goflag.FlagSet)) {
	gofs := goflag.NewFlagSet("", goflag.ContinueOnError)
	install(gofs)
	flags.AddGoFlagSet(gofs)
}

type goValue struct {
	goflag.Value
}

func (goValue) Type() string { return "value" }

// GoValue wraps a flag.Value for a pflag set, which additionally wants Type().
func GoValue(v goflag.Value) pflag.Value {
	return goValue{v}
}

// ErrUsage marks an option-parsing error, which exits 2 like the flag package
// did rather than 1.
type ErrUsage struct{ err error }

func (e ErrUsage) Error() string { return e.err.Error() }
func (e ErrUsage) Unwrap() error { return e.err }

// ParseCobra parses args with the command's own cobra command and returns the
// positional arguments.  Errors come back to the caller instead of cobra
// printing its own message and usage tree.
func ParseCobra(cmd Subcommand, args []string) ([]string, error) {
	rest, _, err := ParseCobraFlags(cmd, args)
	return rest, err
}

// ParseCobraFlags is ParseCobra, additionally handing back the flag set it
// parsed into.  Commands need it to tell an explicitly-passed option from one
// left at its default value, via flags.Changed().
func ParseCobraFlags(cmd Subcommand, args []string) ([]string, *pflag.FlagSet, error) {
	c := cmd.CobraCommand()
	c.SilenceUsage = true
	c.SilenceErrors = true

	flags := c.Flags()
	flags.SetInterspersed(false)

	rewritten, err := SingleDash(flags, args)
	if err != nil {
		return nil, nil, ErrUsage{err}
	}
	if err := flags.Parse(rewritten); err != nil {
		return nil, nil, ErrUsage{err}
	}

	return flags.Args(), flags, nil
}

type CmdFactory func() Subcommand
type subcmd struct {
	args    []string
	nargs   int
	flags   CommandFlags
	factory CmdFactory
}

var subcommands []subcmd = make([]subcmd, 0)

func Register(factory CmdFactory, flags CommandFlags, args ...string) {
	if len(args) == 0 {
		panic("can't register commands with zero arguments")
	}

	subcommands = append(subcommands, subcmd{
		args:    args,
		nargs:   len(args),
		flags:   flags,
		factory: factory,
	})
}

func Lookup(arguments []string) (Subcommand, []string, []string) {
	nargs := len(arguments)
	for _, subcmd := range subcommands {
		if nargs < subcmd.nargs {
			continue
		}

		if !slices.Equal(subcmd.args, arguments[:subcmd.nargs]) {
			continue
		}

		cmd := subcmd.factory()
		cmd.setFlags(subcmd.flags)
		return cmd, arguments[:subcmd.nargs], arguments[subcmd.nargs:]
	}

	return nil, nil, arguments
}

// annotationPath records which registered command a leaf of the tree stands for.
const annotationPath = "plakar.command"

// Resolve returns the command the arguments name and what is left for it.
// Cobra matches the longest path, so "diag snapshot" wins over "diag" without
// registration order having to arrange it.
func Resolve(root *cobra.Command, args []string) (Subcommand, []string, []string) {
	c, rest, err := root.Find(args)
	if err != nil || c == root {
		return nil, nil, args
	}

	// A parent nobody registered ("token" on its own) is not a command.
	path, ok := c.Annotations[annotationPath]
	if !ok {
		return nil, nil, args
	}

	for _, sub := range subcommands {
		if strings.Join(sub.args, " ") != path {
			continue
		}
		cmd := sub.factory()
		cmd.setFlags(sub.flags)
		return cmd, sub.args, rest
	}

	return nil, nil, args
}

// Tree hangs the registered commands off root: "diag snapshot" becomes a child
// of "diag", and an intermediate nobody registered gets a bare parent.
func Tree(root *cobra.Command) {
	byPath := make(map[string]*cobra.Command)

	// Shortest paths first, so a parent exists before its children.
	ordered := make([]subcmd, len(subcommands))
	copy(ordered, subcommands)
	slices.SortStableFunc(ordered, func(a, b subcmd) int {
		return a.nargs - b.nargs
	})

	for _, sub := range ordered {
		parent := root
		for i, name := range sub.args {
			path := strings.Join(sub.args[:i+1], " ")

			if c, ok := byPath[path]; ok {
				parent = c
				continue
			}

			var c *cobra.Command
			leaf := i == len(sub.args)-1
			if leaf {
				cmd := sub.factory()
				cmd.setFlags(sub.flags)
				c = cmd.CobraCommand()
			} else {
				c = &cobra.Command{}
			}
			if leaf {
				c.Annotations = map[string]string{annotationPath: path}
				// Cobra only completes commands it considers
				// runnable; entryPoint does the dispatching.
				c.RunE = func(*cobra.Command, []string) error {
					return nil
				}
			}
			// The name comes from the registration: Use does not
			// always agree ("diag blobsearch" says "diag packfile").
			c.Use = name + argSpec(c.Use, i+1)

			parent.AddCommand(c)
			byPath[path] = c
			parent = c
		}
	}
}

// argSpec returns what a Use string says past its command words:
// "diag blob type mac" with words=2 gives " type mac".
func argSpec(use string, words int) string {
	for range words {
		i := strings.IndexByte(use, ' ')
		if i < 0 {
			return ""
		}
		use = use[i+1:]
	}
	if use == "" {
		return ""
	}
	return " " + use
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
