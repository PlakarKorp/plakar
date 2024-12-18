package header

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type Importer struct {
	Type      string `msgpack:"type" json:"type"`
	Origin    string `msgpack:"origin" json:"origin"`
	Directory string `msgpack:"directory" json:"directory"`
}

type Identity struct {
	Identifier uuid.UUID `msgpack:"identifier" json:"identifier"`
	PublicKey  []byte    `msgpack:"public_key" json:"public_key"`
}

type Class struct {
	Name        string
	Probability float64
}

type Classification struct {
	Analyzer string
	Classes  []string
}

type KeyValue struct {
	Key   string `msgpack:"key" json:"key"`
	Value string `msgpack:"value" json:"value"`
}

type Header struct {
	Identifier      objects.Checksum `msgpack:"identifier" json:"identifier"`
	Version         string           `msgpack:"version" json:"version"`
	Timestamp       time.Time        `msgpack:"timestamp" json:"timestamp"`
	Duration        time.Duration    `msgpack:"duration" json:"duration"`
	Identity        Identity         `msgpack:"identity" json:"identity"`
	Name            string           `msgpack:"name" json:"name"`
	Category        string           `msgpack:"category" json:"category"`
	Environment     string           `msgpack:"environment" json:"environment"`
	Perimeter       string           `msgpack:"perimeter" json:"perimeter"`
	Classifications []Classification `msgpack:"classifications" json:"classifications"`
	Tags            []string         `msgpack:"tags" json:"tags"`
	Context         []KeyValue       `msgpack:"context" json:"context"`
	Importer        Importer         `msgpack:"importer" json:"importer"`
	Root            objects.Checksum `msgpack:"root" json:"root"`
	Errors          objects.Checksum `msgpack:"errors" json:"errors"`
	Index           objects.Checksum `msgpack:"index" json:"index"`
	Metadata        objects.Checksum `msgpack:"metadata" json:"metadata"`
	Statistics      objects.Checksum `msgpack:"statistics" json:"statistics"`
	Summary         vfs.Summary      `msgpack:"summary" json:"summary"`
}

func NewHeader(name string, identifier objects.Checksum) *Header {
	return &Header{
		Identifier:      identifier,
		Timestamp:       time.Now(),
		Version:         storage.VERSION,
		Name:            name,
		Category:        "default",
		Environment:     "default",
		Perimeter:       "default",
		Classifications: []Classification{},
		Tags:            []string{},

		Identity: Identity{},

		Importer: Importer{},

		Context: make([]KeyValue, 0),

		Root:       objects.Checksum{},
		Index:      objects.Checksum{},
		Metadata:   objects.Checksum{},
		Statistics: objects.Checksum{},
	}
}

func NewFromBytes(serialized []byte) (*Header, error) {
	var header Header
	if err := msgpack.Unmarshal(serialized, &header); err != nil {
		return nil, err
	} else {
		return &header, nil
	}
}

func (h *Header) Serialize() ([]byte, error) {
	if serialized, err := msgpack.Marshal(h); err != nil {
		return nil, err
	} else {
		return serialized, nil
	}
}

func (h *Header) SetContext(key, value string) {
	h.Context = append(h.Context, KeyValue{Key: key, Value: value})
}

func (h *Header) GetContext(key string) string {
	for _, kv := range h.Context {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

func (h *Header) GetIndexID() [32]byte {
	return h.Identifier
}

func (h *Header) GetIndexShortID() []byte {
	return h.Identifier[:4]
}

func (h *Header) GetRoot() [32]byte {
	return h.Root
}

func ParseSortKeys(sortKeysStr string) ([]string, error) {
	if sortKeysStr == "" {
		return nil, nil
	}
	keys := strings.Split(sortKeysStr, ",")
	uniqueKeys := make(map[string]bool)
	validKeys := []string{}

	headerType := reflect.TypeOf(Header{})

	for _, key := range keys {
		key = strings.TrimSpace(key)
		lookupKey := key
		if strings.HasPrefix(key, "-") {
			lookupKey = key[1:]
		}
		if uniqueKeys[lookupKey] {
			return nil, errors.New("duplicate sort key: " + key)
		}
		uniqueKeys[lookupKey] = true

		if _, found := headerType.FieldByName(lookupKey); !found {
			return nil, errors.New("invalid sort key: " + key)
		}
		validKeys = append(validKeys, key)
	}

	return validKeys, nil
}

func SortHeaders(headers []Header, sortKeys []string) error {
	var err error
	sort.Slice(headers, func(i, j int) bool {
		for _, key := range sortKeys {
			switch key {
			case "Timestamp":
				if !headers[i].Timestamp.Equal(headers[j].Timestamp) {
					return headers[i].Timestamp.Before(headers[j].Timestamp)
				}
			case "-Timestamp":
				if !headers[i].Timestamp.Equal(headers[j].Timestamp) {
					return headers[i].Timestamp.After(headers[j].Timestamp)
				}
			case "Identifier":
				for k := 0; k < len(headers[i].Identifier); k++ {
					if headers[i].Identifier[k] != headers[j].Identifier[k] {
						return headers[i].Identifier[k] < headers[j].Identifier[k]
					}
				}
			case "-Identifier":
				for k := 0; k < len(headers[i].Identifier); k++ {
					if headers[i].Identifier[k] != headers[j].Identifier[k] {
						return headers[i].Identifier[k] > headers[j].Identifier[k]
					}
				}
			case "Version":
				if headers[i].Version != headers[j].Version {
					return headers[i].Version < headers[j].Version
				}
			case "-Version":
				if headers[i].Version != headers[j].Version {
					return headers[i].Version > headers[j].Version
				}

			case "Tags":
				// Compare Tags lexicographically, element by element
				for k := 0; k < len(headers[i].Tags) && k < len(headers[j].Tags); k++ {
					if headers[i].Tags[k] != headers[j].Tags[k] {
						return headers[i].Tags[k] < headers[j].Tags[k]
					}
				}
				if len(headers[i].Tags) != len(headers[j].Tags) {
					return len(headers[i].Tags) < len(headers[j].Tags)
				}
			case "-Tags":
				// Compare Tags lexicographically, element by element
				for k := 0; k < len(headers[i].Tags) && k < len(headers[j].Tags); k++ {
					if headers[i].Tags[k] != headers[j].Tags[k] {
						return headers[i].Tags[k] > headers[j].Tags[k]
					}
				}
				if len(headers[i].Tags) != len(headers[j].Tags) {
					return len(headers[i].Tags) > len(headers[j].Tags)
				}
			default:
				err = errors.New("invalid sort key: " + key)
				return false
			}
		}
		return false
	})
	return err
}
