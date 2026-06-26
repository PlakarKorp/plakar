package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptsFlagStringNil(t *testing.T) {
	o := NewOptsFlag(nil)
	require.Equal(t, "", o.String())
}

func TestOptsFlagSetAndString(t *testing.T) {
	o := NewOptsFlag(map[string]string{})

	// key=value form
	require.NoError(t, o.Set("foo=bar"))
	// key without '=' defaults to "true"
	require.NoError(t, o.Set("flag"))
	// value containing '=' keeps everything after the first '='
	require.NoError(t, o.Set("kv=a=b"))

	s := o.String()
	// Map iteration order is non-deterministic; assert on the parts.
	require.Contains(t, s, "foo=bar")
	require.Contains(t, s, "flag=true")
	require.Contains(t, s, "kv=a=b")

	// Entries are space-separated; with 3 entries we expect 2 spaces.
	require.Equal(t, 2, strings.Count(s, " "))
}

func TestOptsFlagSetOverwrites(t *testing.T) {
	m := map[string]string{}
	o := NewOptsFlag(m)
	require.NoError(t, o.Set("k=1"))
	require.NoError(t, o.Set("k=2"))
	require.Equal(t, "2", m["k"])
	require.Equal(t, "k=2", o.String())
}
