package main

import (
	"fmt"
	"strings"

	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type defaults struct {
	configDir string
	cacheDir  string
	dataDir   string
	cpuCount  int
}

type globalOpts struct {
	config     string // deprecated, to be removed soon
	configDir  string
	cacheDir   string
	dataDir    string
	cpuCount   int
	maxConc    int
	cpuProfile string
	memProfile string
	time       bool
	trace      string
	json       bool
	stdio      bool
	quiet      bool
	silent     bool
	keyfile    string

	enableSecurityCheck  bool
	disableSecurityCheck bool
}

// singleDashNormalize keeps -cachedir and --cachedir resolving to the same
// flag; the rewriting itself happens in normalizeArgs.
func singleDashNormalize(f *pflag.FlagSet, name string) pflag.NormalizedName {
	return pflag.NormalizedName(strings.ReplaceAll(name, "_", "-"))
}

func lookupLong(root *cobra.Command, name string) *pflag.Flag {
	if f := root.PersistentFlags().Lookup(name); f != nil {
		return f
	}
	return root.Flags().Lookup(name)
}

// takesValue reports whether f consumes the next argument: `-quiet backup`
// must leave `backup` alone.
func takesValue(f *pflag.Flag) bool {
	return f.Value.Type() != "bool"
}

// normalizeArgs rewrites our command line into one pflag can parse: it spells
// the single-dash options with two dashes and pulls out the "at REPOSITORY"
// prefix, which cobra has no notion of.
//
// out is the global part, rest is everything from the subcommand name onwards
// and is never rewritten: the subcommand parses its own options.
func normalizeArgs(root *cobra.Command, args []string) (out, rest []string, repository string, hadAt bool, err error) {
	out = make([]string, 0, len(args))

	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "--" {
			return out, args[i+1:], repository, hadAt, nil
		}

		// The "at REPOSITORY" prefix, only before the subcommand.
		if arg == "at" && !hadAt {
			hadAt = true
			if i+1 < len(args) {
				repository = args[i+1]
				i += 2
			} else {
				i++ // the caller reports the missing repository
			}
			continue
		}

		// A single-dash long option: -cachedir, -cachedir=X, -quiet.
		if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' {
			name, _, explicit := strings.Cut(arg[1:], "=")
			f := lookupLong(root, name)
			if f == nil {
				// Not one of ours: say so, rather than letting
				// pflag complain about a shorthand letter.
				return nil, nil, "", false,
					fmt.Errorf("flag provided but not defined: -%s", name)
			}
			out = append(out, "-"+arg)
			i++
			// `-cachedir X`: carry the value over, or it looks
			// like the subcommand name.
			if !explicit && takesValue(f) && i < len(args) {
				out = append(out, args[i])
				i++
			}
			continue
		}

		if strings.HasPrefix(arg, "--") && len(arg) > 2 {
			name, _, explicit := strings.Cut(arg[2:], "=")
			if f := lookupLong(root, name); f != nil {
				out = append(out, arg)
				i++
				if !explicit && takesValue(f) && i < len(args) {
					out = append(out, args[i])
					i++
				}
				continue
			}
		}

		// Not a flag: this is the subcommand name.
		if !strings.HasPrefix(arg, "-") {
			return out, args[i:], repository, hadAt, nil
		}

		out = append(out, arg)
		i++
	}

	return out, nil, repository, hadAt, nil
}

// isCompletionRequest reports whether the command line is one of the shell
// completion entry points: the hidden command the shells call to ask for
// candidates, or "plakar completion <shell>" which prints the script.
func isCompletionRequest(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return args[0] == cobra.ShellCompRequestCmd ||
		args[0] == cobra.ShellCompNoDescRequestCmd ||
		args[0] == "completion"
}

// completionArgs drops the "at REPOSITORY" prefix from what the shell hands
// back, since cobra resolves commands against the tree and "at" is not one of
// them.  The repository is still read from os.Args when we need it.
//
// The last argument is the word being completed and is left alone: while the
// user is still typing the repository itself there is nothing to strip yet.
func completionArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	last := len(args) - 1

	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "at" && i+1 < last {
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out
}

// newRootCmd builds the root command and binds the global flags.  The setup
// stays in entryPoint() rather than a PersistentPreRun hook: parts of it exit
// early, and the ordering is easier to follow spelled out.
func newRootCmd(opts *globalOpts, def defaults) *cobra.Command {
	root := &cobra.Command{
		Use:              "plakar [OPTIONS] [at REPOSITORY] COMMAND [COMMAND_OPTIONS]...",
		Short:            "effortless backups",
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: false,
	}

	root.SetGlobalNormalizationFunc(singleDashNormalize)

	f := root.PersistentFlags()
	f.StringVar(&opts.config, "config", def.configDir, "configuration directory (deprecated, use -configdir instead)")
	f.StringVar(&opts.configDir, "configdir", def.configDir, "configuration directory")
	f.StringVar(&opts.cacheDir, "cachedir", def.cacheDir, "cache directory")
	f.StringVar(&opts.dataDir, "datadir", def.dataDir, "data directory")
	f.IntVar(&opts.cpuCount, "cpu", def.cpuCount, "limit the number of usable cores")
	f.IntVar(&opts.maxConc, "concurrency", -1, "limit the number of concurrent operations")
	f.StringVar(&opts.cpuProfile, "profile-cpu", "", "profile CPU usage")
	f.StringVar(&opts.memProfile, "profile-mem", "", "profile MEM usage")
	f.BoolVar(&opts.time, "time", false, "display command execution time")
	f.StringVar(&opts.trace, "trace", "", "display trace logs, comma-separated (all, trace, repository, snapshot, server)")
	f.BoolVar(&opts.json, "json", false, "output events as JSON lines")
	f.BoolVar(&opts.stdio, "stdio", false, "use stdio user interface")
	f.BoolVar(&opts.quiet, "quiet", false, "no output except errors")
	f.BoolVar(&opts.silent, "silent", false, "no output at all")
	f.StringVar(&opts.keyfile, "keyfile", "", "use passphrase from key file when prompted")
	f.BoolVar(&opts.enableSecurityCheck, "enable-security-check", false, "enable update check")
	f.BoolVar(&opts.disableSecurityCheck, "disable-security-check", false, "disable update check")

	subcommands.Tree(root)
	installSnapshotCompletion(root)

	root.SetUsageFunc(func(c *cobra.Command) error {
		out := c.OutOrStderr()
		fmt.Fprintf(out, "Usage: %s [OPTIONS] [at REPOSITORY] COMMAND [COMMAND_OPTIONS]...\n", c.Name())
		fmt.Fprintf(out, "\nBy default, the repository is $PLAKAR_REPOSITORY or $HOME/.plakar.\n")
		fmt.Fprintf(out, "\nOPTIONS:\n")
		fmt.Fprint(out, c.PersistentFlags().FlagUsages())
		fmt.Fprintf(out, "\nCOMMANDS:\n")
		listCmds(out, "  ")
		fmt.Fprintf(out, "\nFor more information on a command, use '%s help COMMAND'.\n", c.Name())
		return nil
	})

	return root
}
