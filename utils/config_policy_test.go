package utils

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/locate"
)

func newTestPolicies() *policiesConfig {
	return &policiesConfig{
		Version:  "v1.0.0",
		Policies: make(map[string]*locate.LocateOptions),
	}
}

func TestPoliciesAddHasRemove(t *testing.T) {
	c := newTestPolicies()
	if c.Has("foo") {
		t.Fatal("Has(foo) should be false before Add")
	}
	c.Add("foo")
	if !c.Has("foo") {
		t.Fatal("Has(foo) should be true after Add")
	}
	c.Remove("foo")
	if c.Has("foo") {
		t.Fatal("Has(foo) should be false after Remove")
	}
}

func TestPoliciesSetStringAndUnset(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "name", "snapshot1"); err != nil {
		t.Fatalf("Set name: %v", err)
	}
	if c.Policies["p"].Filters.Name != "snapshot1" {
		t.Fatalf("Filters.Name = %q", c.Policies["p"].Filters.Name)
	}
	if err := c.Unset("p", "name"); err != nil {
		t.Fatalf("Unset name: %v", err)
	}
	if c.Policies["p"].Filters.Name != "" {
		t.Fatalf("Filters.Name after Unset = %q, want empty", c.Policies["p"].Filters.Name)
	}
}

func TestPoliciesSetIntPositive(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "days", "7"); err != nil {
		t.Fatalf("Set days: %v", err)
	}
	if c.Policies["p"].Periods.Day.Keep != 7 {
		t.Fatalf("Periods.Day.Keep = %d, want 7", c.Policies["p"].Periods.Day.Keep)
	}
}

func TestPoliciesSetIntInvalid(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "days", "not-a-number"); err == nil {
		t.Fatal("expected error for non-numeric days, got nil")
	}
}

func TestPoliciesSetIntNegative(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "days", "-1"); err == nil {
		t.Fatal("expected error for negative days, got nil")
	}
}

func TestPoliciesSetStringList(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "tags", "a,b,c"); err != nil {
		t.Fatalf("Set tags: %v", err)
	}
	tags := c.Policies["p"].Filters.Tags
	if len(tags) != 3 || tags[0] != "a" || tags[2] != "c" {
		t.Fatalf("Filters.Tags = %v, want [a b c]", tags)
	}
}

func TestPoliciesSetBool(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "latest", "true"); err != nil {
		t.Fatalf("Set latest: %v", err)
	}
	if !c.Policies["p"].Filters.Latest {
		t.Fatal("Filters.Latest should be true")
	}
	if err := c.Set("p", "latest", "not-bool"); err == nil {
		t.Fatal("expected error for invalid bool")
	}
}

func TestPoliciesSetTime(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	// locate.ParseTimeFlag accepts RFC3339 timestamps.
	if err := c.Set("p", "before", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("Set before: %v", err)
	}
	if c.Policies["p"].Filters.Before.IsZero() {
		t.Fatal("Filters.Before should be set")
	}
	if err := c.Set("p", "before", "not-a-time"); err == nil {
		t.Fatal("expected error for invalid time")
	}
}

func TestPoliciesSetUnknownKey(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "nonsense", "x"); err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
}

func TestPoliciesSetUnknownPolicy(t *testing.T) {
	c := newTestPolicies()
	if err := c.Set("ghost", "name", "x"); err == nil {
		t.Fatal("expected error for missing policy, got nil")
	}
}

func TestPoliciesUnsetUnknownKey(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Unset("p", "nonsense"); err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
}

func TestPoliciesUnsetAllTypes(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	_ = c.Set("p", "name", "x")
	_ = c.Set("p", "tags", "a,b")
	_ = c.Set("p", "latest", "true")
	_ = c.Set("p", "days", "3")
	_ = c.Set("p", "before", "2026-01-01T00:00:00Z")
	for _, k := range []string{"name", "tags", "latest", "days", "before"} {
		if err := c.Unset("p", k); err != nil {
			t.Fatalf("Unset %s: %v", k, err)
		}
	}
	p := c.Policies["p"]
	if p.Filters.Name != "" || p.Filters.Tags != nil || p.Filters.Latest || p.Periods.Day.Keep != 0 || !p.Filters.Before.IsZero() {
		t.Fatalf("expected all zeroed, got %+v", p)
	}
}

func TestPoliciesDumpJSON(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	_ = c.Set("p", "name", "snap")
	var buf bytes.Buffer
	if err := c.Dump(&buf, "json", []string{"p"}); err != nil {
		t.Fatalf("Dump json: %v", err)
	}
	var got map[string]map[string]any
	if err := json.NewDecoder(&buf).Decode(&got); err != nil {
		t.Fatalf("decode dumped json: %v", err)
	}
	if _, has := got["p"]; !has {
		t.Fatalf("dump missing 'p': %+v", got)
	}
}

func TestPoliciesDumpYAML(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	var buf bytes.Buffer
	if err := c.Dump(&buf, "yaml", []string{"p"}); err != nil {
		t.Fatalf("Dump yaml: %v", err)
	}
	if !strings.Contains(buf.String(), "p:") {
		t.Fatalf("dump did not contain key marker: %q", buf.String())
	}
}

func TestPoliciesDumpAllWhenNoNames(t *testing.T) {
	c := newTestPolicies()
	c.Add("a")
	c.Add("b")
	var buf bytes.Buffer
	if err := c.Dump(&buf, "yaml", nil); err != nil {
		t.Fatalf("Dump yaml: %v", err)
	}
	if !strings.Contains(buf.String(), "a:") || !strings.Contains(buf.String(), "b:") {
		t.Fatalf("expected both policies dumped, got: %q", buf.String())
	}
}

func TestPoliciesDumpMissingEntry(t *testing.T) {
	c := newTestPolicies()
	if err := c.Dump(&bytes.Buffer{}, "json", []string{"missing"}); err == nil {
		t.Fatal("expected error for missing entry, got nil")
	}
}

func TestPoliciesDumpUnknownFormat(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Dump(&bytes.Buffer{}, "xml", []string{"p"}); err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}

func TestPoliciesSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policies.yml")

	c := newTestPolicies()
	c.Add("nightly")
	if err := c.Set("nightly", "days", "30"); err != nil {
		t.Fatalf("Set days: %v", err)
	}
	if err := c.Set("nightly", "tags", "auto,daily"); err != nil {
		t.Fatalf("Set tags: %v", err)
	}
	if err := c.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	loaded, err := LoadPolicyConfigFile(path)
	if err != nil {
		t.Fatalf("LoadPolicyConfigFile: %v", err)
	}
	if !loaded.Has("nightly") {
		t.Fatal("loaded config missing 'nightly'")
	}
	if loaded.Policies["nightly"].Periods.Day.Keep != 30 {
		t.Fatalf("days = %d", loaded.Policies["nightly"].Periods.Day.Keep)
	}
	if len(loaded.Policies["nightly"].Filters.Tags) != 2 {
		t.Fatalf("tags = %v", loaded.Policies["nightly"].Filters.Tags)
	}
}

func TestLoadPolicyConfigFileMissingReturnsEmptyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.yml")
	cfg, err := LoadPolicyConfigFile(path)
	if err != nil {
		t.Fatalf("LoadPolicyConfigFile: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if len(cfg.Policies) != 0 {
		t.Fatalf("expected empty Policies, got %v", cfg.Policies)
	}
}

func TestLoadPolicyConfigFileBadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policies.yml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadPolicyConfigFile(path); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestPoliciesApplyConfigCopiesOptions(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	_ = c.Set("p", "name", "snap")
	var dst locate.LocateOptions
	c.ApplyConfig("p", &dst)
	if dst.Filters.Name != "snap" {
		t.Fatalf("ApplyConfig did not copy: %+v", dst.Filters)
	}
}

func TestPoliciesApplyConfigUnknownIsNoOp(t *testing.T) {
	c := newTestPolicies()
	dst := locate.LocateOptions{}
	dst.Filters.Name = "preserved"
	c.ApplyConfig("ghost", &dst)
	if dst.Filters.Name != "preserved" {
		t.Fatal("ApplyConfig with unknown name should not overwrite")
	}
}

// TestPoliciesLocateFieldAllStringKeys walks every string-valued filter
// key supported by locateField, exercising each switch branch once.
func TestPoliciesLocateFieldAllStringKeys(t *testing.T) {
	stringKeys := []string{
		"name", "category", "environment", "perimeter", "job",
	}
	for _, key := range stringKeys {
		t.Run(key, func(t *testing.T) {
			c := newTestPolicies()
			c.Add("p")
			if err := c.Set("p", key, "value-for-"+key); err != nil {
				t.Fatalf("Set %s: %v", key, err)
			}
			if err := c.Unset("p", key); err != nil {
				t.Fatalf("Unset %s: %v", key, err)
			}
		})
	}
}

// TestPoliciesLocateFieldAllStringListKeys exercises the []string branches.
func TestPoliciesLocateFieldAllStringListKeys(t *testing.T) {
	listKeys := []string{"tags", "ids", "roots"}
	for _, key := range listKeys {
		t.Run(key, func(t *testing.T) {
			c := newTestPolicies()
			c.Add("p")
			if err := c.Set("p", key, "a,b,c"); err != nil {
				t.Fatalf("Set %s: %v", key, err)
			}
			if err := c.Unset("p", key); err != nil {
				t.Fatalf("Unset %s: %v", key, err)
			}
		})
	}
}

// TestPoliciesLocateFieldAllIntKeys exercises every period-related int branch
// (Keep and Cap for each period). Each branch is hit via both Set and Unset.
func TestPoliciesLocateFieldAllIntKeys(t *testing.T) {
	intKeys := []string{
		// Periods.*.Keep
		"minutes", "hours", "days", "weeks", "months", "years",
		"mondays", "tuesdays", "wednesdays", "thursdays",
		"fridays", "saturdays", "sundays",
		// Periods.*.Cap
		"per-minute", "per-hour", "per-day", "per-week", "per-month", "per-year",
		"per-monday", "per-tuesday", "per-wednesday", "per-thursday",
		"per-friday", "per-saturday", "per-sunday",
	}
	for _, key := range intKeys {
		t.Run(key, func(t *testing.T) {
			c := newTestPolicies()
			c.Add("p")
			if err := c.Set("p", key, "5"); err != nil {
				t.Fatalf("Set %s: %v", key, err)
			}
			if err := c.Unset("p", key); err != nil {
				t.Fatalf("Unset %s: %v", key, err)
			}
		})
	}
}

// TestPoliciesSetKeepAndCapPerPeriod pins `<period>s` to Keep and
// `per-<period>` to Cap: `policy set weekly per-week=1` must cap each week at
// one snapshot, not keep a single week, or prune deletes the older ones.
func TestPoliciesSetKeepAndCapPerPeriod(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	p := &c.Policies["p"].Periods
	for _, tc := range []struct {
		name   string
		period *locate.LocatePeriod
	}{
		{"minute", &p.Minute}, {"hour", &p.Hour}, {"day", &p.Day},
		{"week", &p.Week}, {"month", &p.Month}, {"year", &p.Year},
		{"monday", &p.Monday}, {"tuesday", &p.Tuesday}, {"wednesday", &p.Wednesday},
		{"thursday", &p.Thursday}, {"friday", &p.Friday}, {"saturday", &p.Saturday},
		{"sunday", &p.Sunday},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := c.Set("p", tc.name+"s", "3"); err != nil {
				t.Fatalf("Set %ss: %v", tc.name, err)
			}
			if err := c.Set("p", "per-"+tc.name, "2"); err != nil {
				t.Fatalf("Set per-%s: %v", tc.name, err)
			}
			if want := (locate.LocatePeriod{Keep: 3, Cap: 2}); *tc.period != want {
				t.Fatalf("%s = %+v, want %+v", tc.name, *tc.period, want)
			}
		})
	}
}

// TestPoliciesLocateFieldSinceAndLatest exercises the last remaining
// non-string/non-int branches: `since` (time.Time) and `latest` (bool).
func TestPoliciesLocateFieldSinceAndLatest(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "since", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("Set since: %v", err)
	}
	if err := c.Unset("p", "since"); err != nil {
		t.Fatalf("Unset since: %v", err)
	}
	if err := c.Set("p", "latest", "true"); err != nil {
		t.Fatalf("Set latest: %v", err)
	}
	if err := c.Unset("p", "latest"); err != nil {
		t.Fatalf("Unset latest: %v", err)
	}
}

// TestPoliciesLocateFieldInvalidKey hits the default branch.
func TestPoliciesLocateFieldInvalidKey(t *testing.T) {
	c := newTestPolicies()
	c.Add("p")
	if err := c.Set("p", "bogus-key", "x"); err == nil {
		t.Fatal("Set with invalid key should error")
	}
	if err := c.Unset("p", "bogus-key"); err == nil {
		t.Fatal("Unset with invalid key should error")
	}
}
