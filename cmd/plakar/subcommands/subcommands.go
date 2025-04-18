package subcommands

import (
	"fmt"
	"sort"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/vmihailenco/msgpack/v5"
)

type CommandFlags uint32

const (
	NeedRepositoryKey CommandFlags = 1 << iota
	BeforeRepositoryOpen
	AgentSupport
)

type Subcommand interface {
	Parse(ctx *appcontext.AppContext, args []string) error
	Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error)
	GetRepositorySecret() []byte
	GetFlags() CommandFlags
}

type SubcommandBase struct {
	RepositorySecret []byte
	Flags            CommandFlags
}

func (cmd *SubcommandBase) GetFlags() CommandFlags {
	return cmd.Flags
}

func (cmd *SubcommandBase) GetRepositorySecret() []byte {
	return cmd.RepositorySecret
}

type subcmd struct {
	args []string
	cmd  Subcommand
}

var subcommands map[string]Subcommand = make(map[string]Subcommand)

func Register(cmd Subcommand, name string) {
	subcommands[name] = cmd
}

func Lookup(name string) Subcommand {
	cmd, ok := subcommands[name]

	if !ok {
		return nil
	} else {
		return cmd
	}
}

func List() []string {
	var list []string
	for command := range subcommands {
		list = append(list, command)
	}
	sort.Strings(list)
	return list
}

// RPC extends subcommands.Subcommand, but it also includes the Name() method used to identify the RPC on decoding.
type RPC interface {
	Subcommand
	Name() string
}

type encodedRPC struct {
	Name        string
	Subcommand  RPC
	StoreConfig map[string]string
}

// Encode marshals the RPC into the msgpack encoder. It prefixes the RPC with
// the Name() of the RPC. This is used to identify the RPC on decoding.
func EncodeRPC(encoder *msgpack.Encoder, cmd RPC, storeConfig map[string]string) error {
	return encoder.Encode(encodedRPC{
		Name:        cmd.Name(),
		Subcommand:  cmd,
		StoreConfig: storeConfig,
	})
}

// Decode extracts the request encoded by Encode(). It returns the name of the
// RPC and the raw bytes of the request. The raw bytes can be used by the caller
// to unmarshal the bytes with the correct struct.
func DecodeRPC(decoder *msgpack.Decoder) (string, map[string]string, []byte, error) {
	var request map[string]interface{}
	if err := decoder.Decode(&request); err != nil {
		return "", nil, nil, fmt.Errorf("failed to decode client request: %w", err)
	}

	rawRequest, err := msgpack.Marshal(request)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to marshal client request: %s", err)
	}

	name, ok := request["Name"].(string)
	if !ok {
		return "", nil, nil, fmt.Errorf("request does not contain a Name string field")
	}

	storeConfig, ok := request["StoreConfig"].(map[string]interface{})
	if !ok {
		return "", nil, nil, fmt.Errorf("request does not contain a StoreConfig field")
	}

	okStoreConfig := make(map[string]string)
	for k, v := range storeConfig {
		if str, ok := v.(string); ok {
			okStoreConfig[k] = str
		} else {
			return "", nil, nil, fmt.Errorf("StoreConfig field %s is not a string", k)
		}
	}

	return name, okStoreConfig, rawRequest, nil
}
