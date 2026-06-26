package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/stretchr/testify/require"
)

func newPolicies() *policiesConfig {
	return &policiesConfig{
		Version:  "v1.0.0",
		Policies: map[string]*locate.LocateOptions{},
	}
}

func TestPolicyHasAddRemove(t *testing.T) {
	c := newPolicies()
	require.False(t, c.Has("daily"))
	c.Add("daily")
	require.True(t, c.Has("daily"))
	require.NotNil(t, c.Policies["daily"])
	c.Remove("daily")
	require.False(t, c.Has("daily"))
}

func TestPolicySetInt(t *testing.T) {
	c := newPolicies()
	c.Add("p")

	require.NoError(t, c.Set("p", "days", "7"))
	require.Equal(t, 7, c.Policies["p"].Periods.Day.Keep)

	require.NoError(t, c.Set("p", "per-hour", "3"))
	require.Equal(t, 3, c.Policies["p"].Periods.Hour.Cap)

	// non-numeric
	err := c.Set("p", "days", "abc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid value")

	// negative
	err = c.Set("p", "days", "-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "negative value")
}

func TestPolicySetString(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	require.NoError(t, c.Set("p", "name", "myname"))
	require.Equal(t, "myname", c.Policies["p"].Filters.Name)

	require.NoError(t, c.Set("p", "category", "cat"))
	require.Equal(t, "cat", c.Policies["p"].Filters.Category)
}

func TestPolicySetStringList(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	require.NoError(t, c.Set("p", "tags", "a,b,c"))
	require.Equal(t, []string{"a", "b", "c"}, c.Policies["p"].Filters.Tags)
}

func TestPolicySetTime(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	require.NoError(t, c.Set("p", "before", "2025-04-15"))
	require.False(t, c.Policies["p"].Filters.Before.IsZero())

	err := c.Set("p", "before", "not-a-date")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid value")
}

func TestPolicySetBool(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	require.NoError(t, c.Set("p", "latest", "true"))
	require.True(t, c.Policies["p"].Filters.Latest)

	err := c.Set("p", "latest", "notbool")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid value")
}

func TestPolicySetUnknownKeyOrEntry(t *testing.T) {
	c := newPolicies()
	c.Add("p")

	err := c.Set("p", "nope", "x")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid key")

	err = c.Set("missing", "days", "1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "entry not found")
}

func TestPolicyUnset(t *testing.T) {
	c := newPolicies()
	c.Add("p")

	require.NoError(t, c.Set("p", "days", "7"))
	require.NoError(t, c.Set("p", "name", "foo"))
	require.NoError(t, c.Set("p", "tags", "a,b"))
	require.NoError(t, c.Set("p", "before", "2025-04-15"))
	require.NoError(t, c.Set("p", "latest", "true"))

	require.NoError(t, c.Unset("p", "days"))
	require.Equal(t, 0, c.Policies["p"].Periods.Day.Keep)

	require.NoError(t, c.Unset("p", "name"))
	require.Equal(t, "", c.Policies["p"].Filters.Name)

	require.NoError(t, c.Unset("p", "tags"))
	require.Nil(t, c.Policies["p"].Filters.Tags)

	require.NoError(t, c.Unset("p", "before"))
	require.True(t, c.Policies["p"].Filters.Before.IsZero())

	require.NoError(t, c.Unset("p", "latest"))
	require.False(t, c.Policies["p"].Filters.Latest)

	err := c.Unset("p", "nope")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid key")
}

func TestPolicyAllLocateFields(t *testing.T) {
	// Exercise every key understood by locateField via Set, so the big
	// switch is fully covered.
	intKeys := []string{
		"minutes", "hours", "days", "weeks", "months", "years",
		"mondays", "tuesdays", "wednesdays", "thursdays", "fridays", "saturdays", "sundays",
		"per-minute", "per-hour", "per-day", "per-week", "per-month", "per-year",
		"per-monday", "per-tuesday", "per-wednesday", "per-thursday", "per-friday",
		"per-saturday", "per-sunday",
	}
	stringKeys := []string{
		"name", "category", "environment", "perimeter", "job",
	}
	listKeys := []string{"tags", "ids", "roots"}
	timeKeys := []string{"before", "since"}

	c := newPolicies()
	c.Add("p")

	for _, k := range intKeys {
		require.NoError(t, c.Set("p", k, "1"), "int key %s", k)
	}
	for _, k := range stringKeys {
		require.NoError(t, c.Set("p", k, "v"), "string key %s", k)
	}
	for _, k := range listKeys {
		require.NoError(t, c.Set("p", k, "a,b"), "list key %s", k)
	}
	for _, k := range timeKeys {
		require.NoError(t, c.Set("p", k, "2025-04-15"), "time key %s", k)
	}
	require.NoError(t, c.Set("p", "latest", "true"))

	require.Equal(t, "v", c.Policies["p"].Filters.Environment)
	require.Equal(t, "v", c.Policies["p"].Filters.Perimeter)
	require.Equal(t, "v", c.Policies["p"].Filters.Job)
	require.Equal(t, []string{"a", "b"}, c.Policies["p"].Filters.IDs)
	require.Equal(t, []string{"a", "b"}, c.Policies["p"].Filters.Roots)
	require.False(t, c.Policies["p"].Filters.Since.IsZero())
	require.Equal(t, 1, c.Policies["p"].Periods.Minute.Keep)
	require.Equal(t, 1, c.Policies["p"].Periods.Month.Cap)
}

func TestPolicyWeekdayKeys(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	require.NoError(t, c.Set("p", "mondays", "2"))
	require.Equal(t, 2, c.Policies["p"].Periods.Monday.Keep)
	require.NoError(t, c.Set("p", "per-sunday", "5"))
	require.Equal(t, 5, c.Policies["p"].Periods.Sunday.Cap)
	require.NoError(t, c.Set("p", "weeks", "4"))
	require.Equal(t, 4, c.Policies["p"].Periods.Week.Keep)
	require.NoError(t, c.Set("p", "per-year", "1"))
	require.Equal(t, 1, c.Policies["p"].Periods.Year.Cap)
}

func TestPolicyDumpJSONAndYAML(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	require.NoError(t, c.Set("p", "days", "7"))

	var jsonBuf bytes.Buffer
	require.NoError(t, c.Dump(&jsonBuf, "json", []string{"p"}))
	require.Contains(t, jsonBuf.String(), "\"p\"")

	var yamlBuf bytes.Buffer
	require.NoError(t, c.Dump(&yamlBuf, "yaml", nil))
	require.Contains(t, yamlBuf.String(), "p:")

	// unknown format
	var b bytes.Buffer
	err := c.Dump(&b, "xml", []string{"p"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown format")

	// unknown entry
	err = c.Dump(&b, "json", []string{"missing"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestPolicySaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "policies.yml")

	c := newPolicies()
	c.Add("daily")
	require.NoError(t, c.Set("daily", "days", "7"))
	require.NoError(t, c.Set("daily", "name", "nightly"))

	require.NoError(t, c.SaveToFile(file))
	_, err := os.Stat(file)
	require.NoError(t, err)

	loaded, err := LoadPolicyConfigFile(file)
	require.NoError(t, err)
	require.True(t, loaded.Has("daily"))
	require.Equal(t, 7, loaded.Policies["daily"].Periods.Day.Keep)
	require.Equal(t, "nightly", loaded.Policies["daily"].Filters.Name)
}

func TestLoadPolicyConfigFileMissing(t *testing.T) {
	// Missing file returns an initialized, empty config.
	c, err := LoadPolicyConfigFile(filepath.Join(t.TempDir(), "nope.yml"))
	require.NoError(t, err)
	require.NotNil(t, c)
	require.Empty(t, c.Policies)
	require.Equal(t, "v1.0.0", c.Version)
}

func TestLoadPolicyConfigFileInvalid(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "bad.yml")
	require.NoError(t, os.WriteFile(file, []byte("policies: [unterminated"), 0600))
	_, err := LoadPolicyConfigFile(file)
	require.Error(t, err)
}

func TestPolicyLoadFromReader(t *testing.T) {
	c := newPolicies()
	data := "version: v1.0.0\npolicies:\n  weekly:\n    Filters:\n      Name: w\n"
	require.NoError(t, c.Load(strings.NewReader(data)))
	require.True(t, c.Has("weekly"))
}

func TestPolicyApplyConfig(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	require.NoError(t, c.Set("p", "days", "9"))

	var po locate.LocateOptions
	c.ApplyConfig("p", &po)
	require.Equal(t, 9, po.Periods.Day.Keep)

	// Applying a missing entry leaves the target untouched.
	var po2 locate.LocateOptions
	c.ApplyConfig("missing", &po2)
	require.Equal(t, 0, po2.Periods.Day.Keep)
}
