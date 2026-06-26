package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadINI(t *testing.T) {
	data := `[remote]
location = s3://bucket
key = value

[other]
foo = bar
`
	res, err := LoadINI(strings.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, "s3://bucket", res["remote"]["location"])
	require.Equal(t, "value", res["remote"]["key"])
	require.Equal(t, "bar", res["other"]["foo"])
}

func TestLoadINIInvalid(t *testing.T) {
	// A line with no section and malformed content -> ini parse error.
	_, err := LoadINI(strings.NewReader("=no-key\n[unterminated"))
	require.Error(t, err)
}

func TestLoadYAML(t *testing.T) {
	data := `remote:
  location: s3://bucket
  count: 3
  enabled: true
scalar: ignored
`
	res, err := LoadYAML(strings.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, "s3://bucket", res["remote"]["location"])
	// non-string scalars are stringified by toString
	require.Equal(t, "3", res["remote"]["count"])
	require.Equal(t, "true", res["remote"]["enabled"])
	// non-object top-level keys are skipped
	_, ok := res["scalar"]
	require.False(t, ok)
}

func TestLoadYAMLInvalid(t *testing.T) {
	_, err := LoadYAML(strings.NewReader("foo: [bar"))
	require.Error(t, err)
}

func TestLoadJSON(t *testing.T) {
	data := `{"remote": {"location": "s3://bucket", "key": "value"}}`
	res, err := LoadJSON(strings.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, "s3://bucket", res["remote"]["location"])
	require.Equal(t, "value", res["remote"]["key"])
}

func TestLoadJSONInvalid(t *testing.T) {
	_, err := LoadJSON(strings.NewReader("{not json"))
	require.Error(t, err)
}

func TestGetConfYAML(t *testing.T) {
	data := `remote:
  location: s3://bucket
  empty:
  key: value
`
	res, err := GetConf(strings.NewReader(data), "")
	require.NoError(t, err)
	require.Equal(t, "s3://bucket", res["remote"]["location"])
	require.Equal(t, "value", res["remote"]["key"])
	// empty values are stripped
	_, ok := res["remote"]["empty"]
	require.False(t, ok)
}

func TestGetConfMissingLocation(t *testing.T) {
	data := `remote:
  key: value
`
	_, err := GetConf(strings.NewReader(data), "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing 'location' key")
}

func TestGetConfThirdParty(t *testing.T) {
	// With a thirdParty set, keys are prefixed and a synthetic location added.
	data := `remote:
  host: example.com
  user: bob
`
	res, err := GetConf(strings.NewReader(data), "rclone")
	require.NoError(t, err)
	require.Equal(t, "rclone://", res["remote"]["location"])
	require.Equal(t, "example.com", res["remote"]["rclone_host"])
	require.Equal(t, "bob", res["remote"]["rclone_user"])
	// original (unprefixed) keys are gone
	_, ok := res["remote"]["host"]
	require.False(t, ok)
}

func TestGetConfINI(t *testing.T) {
	// INI parsing is the last fallback after YAML and JSON both fail to
	// produce a usable structure.
	data := "[remote]\nlocation = s3://bucket\nkey = value\n"
	res, err := GetConf(strings.NewReader(data), "")
	require.NoError(t, err)
	require.Equal(t, "s3://bucket", res["remote"]["location"])
	require.Equal(t, "value", res["remote"]["key"])
}
