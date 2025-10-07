package subcommands

import (
	"fmt"
	"slices"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/vmihailenco/msgpack/v5"
)

type CommandFlags uint32

const (
	NeedRepositoryKey CommandFlags = 1 << iota
	BeforeRepositoryWithStorage
	BeforeRepositoryOpen
	AgentSupport
	IgnoreVersion
)

type Subcommand interface {
	Parse(ctx *appcontext.AppContext, args []string) error
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
	GetRepositorySecret() []byte
	GetFlags() CommandFlags
	setFlags(CommandFlags)
	GetCWD() string
	SetCWD(string)
	GetCommandLine() string
	SetCommandLine(string)

	GetLogInfo() bool
	SetLogInfo(bool)
	GetLogTraces() string
	SetLogTraces(string)
}

type SubcommandBase struct {
	RepositorySecret []byte
	Flags            CommandFlags
	CWD              string
	CommandLine      string

	// XXX - rework that post-release
	LogInfo   bool
	LogTraces string
}

func (cmd *SubcommandBase) setFlags(flags CommandFlags) {
	cmd.Flags = flags
}

func (cmd *SubcommandBase) GetFlags() CommandFlags {
	return cmd.Flags
}

func (cmd *SubcommandBase) GetCWD() string {
	return cmd.CWD
}

func (cmd *SubcommandBase) SetCWD(cwd string) {
	cmd.CWD = cwd
}

func (cmd *SubcommandBase) GetCommandLine() string {
	return cmd.CommandLine
}

func (cmd *SubcommandBase) SetCommandLine(cmdline string) {
	cmd.CommandLine = cmdline
}

func (cmd *SubcommandBase) GetLogInfo() bool {
	return cmd.LogInfo
}

func (cmd *SubcommandBase) GetLogTraces() string {
	return cmd.LogTraces
}

func (cmd *SubcommandBase) SetLogInfo(v bool) {
	cmd.LogInfo = v
}

func (cmd *SubcommandBase) SetLogTraces(traces string) {
	cmd.LogTraces = traces
}

func (cmd *SubcommandBase) GetRepositorySecret() []byte {
	return cmd.RepositorySecret
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

type encodedRPC struct {
	Name        []string
	Subcommand  Subcommand
	StoreConfig map[string]string
}

// Encode marshals the RPC into the msgpack encoder. It prefixes the RPC with
// the Name() of the RPC. This is used to identify the RPC on decoding.
func EncodeRPC(encoder *msgpack.Encoder, name []string, cmd Subcommand, storeConfig map[string]string) error {
	return encoder.Encode(encodedRPC{
		Name:        name,
		Subcommand:  cmd,
		StoreConfig: storeConfig,
	})
}

// Decode extracts the request encoded by Encode(). It returns the name of the
// RPC and the raw bytes of the request. The raw bytes can be used by the caller
// to unmarshal the bytes with the correct struct.
func DecodeRPC(decoder *msgpack.Decoder) ([]string, map[string]string, []byte, error) {
	var request map[string]interface{}
	if err := decoder.Decode(&request); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode client request: %w", err)
	}

	subcommand, exists := request["Subcommand"]
	if !exists {
		return nil, nil, nil, fmt.Errorf("request does not contain a Subcommand field")
	}
	rawRequest, err := msgpack.Marshal(subcommand)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to marshal client request: %w", err)
	}

	nameRaw, exists := request["Name"]
	if !exists {
		return nil, nil, nil, fmt.Errorf("request does not contain a Name field")
	}
	tmp, ok := nameRaw.([]interface{})
	if !ok {
		return nil, nil, nil, fmt.Errorf("request Name field is not an array")
	}

	name := make([]string, 0, len(tmp))
	for i, elm := range tmp {
		str, ok := elm.(string)
		if !ok {
			return nil, nil, nil, fmt.Errorf("request Name field element %d is not a string", i)
		}
		name = append(name, str)
	}

	storeConfigRaw, exists := request["StoreConfig"]
	if !exists {
		return nil, nil, nil, fmt.Errorf("request does not contain a StoreConfig field")
	}
	storeConfig, ok := storeConfigRaw.(map[string]interface{})
	if !ok {
		return nil, nil, nil, fmt.Errorf("request StoreConfig field is not a map")
	}

	okStoreConfig := make(map[string]string)
	for k, v := range storeConfig {
		str, ok := v.(string)
		if !ok {
			return nil, nil, nil, fmt.Errorf("StoreConfig field %s is not a string", k)
		}
		okStoreConfig[k] = str
	}

	return name, okStoreConfig, rawRequest, nil
}
