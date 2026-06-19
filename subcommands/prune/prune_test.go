package prune

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateRepoAndTwoSnaps(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *snapshot.Snapshot, *appcontext.AppContext) {
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	// First snapshot
	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/a.txt", 0644, "hello A"),
	})

	// Second snapshot (newest)
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/b.txt", 0644, "hello B"),
	})

	return repo, snap1, snap2, ctx
}

func TestPrune_DryRun_PerMinuteCap(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap1, snap2, ctx := generateRepoAndTwoSnaps(t, bufOut, bufErr)
	defer snap1.Close()
	defer snap2.Close()

	// Cap 1 per minute across all minute buckets. With two snaps in the same minute,
	// prune will keep the newest and mark the older for delete — but dry-run prints a plan only.
	args := []string{"--per-minute=1"}

	cmd := &Prune{}
	err := cmd.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()

	// In dry-run, we get a summary like:
	//   prune: would keep X and delete Y snapshot(s), run with -apply to proceed
	require.Contains(t, out, "prune: would keep 1 and delete 1 snapshot(s)")
	// Should list minute matches/caps
	require.Contains(t, out, "match=minute:")
	require.Contains(t, out, "cap=1")
	// Should NOT have the actual removal line without -apply
	require.NotContains(t, out, "prune: removal of")
}

func TestPrune_Apply_PerMinuteCap(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap1, snap2, ctx := generateRepoAndTwoSnaps(t, bufOut, bufErr)
	defer snap1.Close()
	defer snap2.Close()

	// With -apply the older snapshot should actually be removed.
	// Retention keeps the newest in the minute; snap1 is older → deleted.
	args := []string{"-apply", "--per-minute=1"}

	cmd := &Prune{}
	err := cmd.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, cmd)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()

	// prune logs via the app context logger:
	//   info: rm: removal of <first 4 bytes hex> completed successfully
	// The "short id" from GetIndexShortID() should match those 4 bytes.
	short1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	require.Contains(t, out, fmt.Sprintf("info: prune: removal of %s completed successfully", short1))

	// Sanity: ensure it didn't claim to remove the newest one (kept)
	short2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	require.NotContains(t, out, fmt.Sprintf("info: prune: removal of %s completed successfully", short2))
}

// TestMergePolicyOptions_PreservesPolicyFiltersWhenCLIEmpty is the regression
// test for https://github.com/PlakarKorp/plakar/issues/1758
//
// Before the fix, mergePolicyOptions did `to.Filters = from.Filters`, which
// overwrote the policy-loaded filters with the (typically empty) CLI override
// struct. Effect: `-policy daily` (where the policy filters by tag) silently
// matched every snapshot.
func TestMergePolicyOptions_PreservesPolicyFiltersWhenCLIEmpty(t *testing.T) {
	policy := locate.NewDefaultLocateOptions()
	policy.Filters.Tags = []string{"daily"}
	policy.Periods.Minute.Keep = 5

	cli := locate.NewDefaultLocateOptions() // user passed no other flags

	mergePolicyOptions(policy, cli)

	require.Equal(t, []string{"daily"}, policy.Filters.Tags,
		"policy filters must survive an empty CLI override")
	require.Equal(t, 5, policy.Periods.Minute.Keep,
		"policy periods must survive an empty CLI override")
}

// TestMergePolicyOptions_CLIOverridesPolicy pins that explicit CLI flags do
// take precedence over policy values.
func TestMergePolicyOptions_CLIOverridesPolicy(t *testing.T) {
	policy := locate.NewDefaultLocateOptions()
	policy.Filters.Tags = []string{"daily"}
	policy.Filters.Name = "policy-name"
	policy.Periods.Minute.Keep = 5

	cli := locate.NewDefaultLocateOptions()
	cli.Filters.Tags = []string{"weekly"}
	cli.Filters.Name = "cli-name"
	cli.Periods.Minute.Keep = 9

	mergePolicyOptions(policy, cli)

	require.Equal(t, []string{"weekly"}, policy.Filters.Tags)
	require.Equal(t, "cli-name", policy.Filters.Name)
	require.Equal(t, 9, policy.Periods.Minute.Keep)
}

// TestMergePolicyOptions_CarriesGroupBy is the regression test for the bug
// where `prune -group-by perimeter` silently applied retention globally.
//
// prune parses its locate flags into a temporary override struct and merges
// them onto the active options via mergePolicyOptions. Before the fix, that
// merge copied Filters and every Period but never copied GroupBy, so GroupBy
// stayed GroupByNone and `-per-day`/`-per-minute` caps were applied across all
// snapshots instead of independently within each group.
func TestMergePolicyOptions_CarriesGroupBy(t *testing.T) {
	// CLI sets group-by; policy has none.
	policy := locate.NewDefaultLocateOptions()
	cli := locate.NewDefaultLocateOptions()
	cli.GroupBy = locate.GroupByPerimeter

	mergePolicyOptions(policy, cli)

	require.Equal(t, locate.GroupByPerimeter, policy.GroupBy,
		"CLI -group-by must be carried into the active options")
}

// TestMergePolicyOptions_GroupByEmptyCLIKeepsPolicy pins that a group-by set in
// a policy survives an empty CLI override (mirrors the filters/periods rule).
func TestMergePolicyOptions_GroupByEmptyCLIKeepsPolicy(t *testing.T) {
	policy := locate.NewDefaultLocateOptions()
	policy.GroupBy = locate.GroupByPerimeter
	cli := locate.NewDefaultLocateOptions() // user passed no -group-by

	mergePolicyOptions(policy, cli)

	require.Equal(t, locate.GroupByPerimeter, policy.GroupBy,
		"policy group-by must survive an empty CLI override")
}

// TestMergeFilters exhaustively covers each field of LocateFilters to make
// sure mergeFilters keeps `a` when `b` is zero, and adopts `b` when it isn't.
func TestMergeFilters(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	t.Run("empty b keeps a", func(t *testing.T) {
		a := locate.LocateFilters{
			Before:      now,
			Since:       now,
			Name:        "n",
			Category:    "c",
			Environment: "e",
			Perimeter:   "p",
			Job:         "j",
			Tags:        []string{"t1"},
			IgnoreTags:  []string{"!t1"},
			Latest:      true,
			IDs:         []string{"id1"},
			Types:       []string{"fs"},
			Origins:     []string{"host"},
			Roots:       []string{"/root"},
		}
		want := a
		b := locate.LocateFilters{}
		mergeFilters(&a, &b)
		if !reflect.DeepEqual(a, want) {
			t.Fatalf("a changed when b was empty:\n got  %#v\n want %#v", a, want)
		}
	})

	t.Run("non-zero b wins", func(t *testing.T) {
		a := locate.LocateFilters{
			Before:      now,
			Name:        "old",
			Tags:        []string{"old"},
			Latest:      false,
		}
		b := locate.LocateFilters{
			Before:      later,
			Since:       later,
			Name:        "new",
			Category:    "c2",
			Environment: "e2",
			Perimeter:   "p2",
			Job:         "j2",
			Tags:        []string{"new"},
			IgnoreTags:  []string{"!new"},
			Latest:      true,
			IDs:         []string{"id2"},
			Types:       []string{"s3"},
			Origins:     []string{"host2"},
			Roots:       []string{"/r2"},
		}
		mergeFilters(&a, &b)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("merge with all-non-zero b should equal b:\n got  %#v\n want %#v", a, b)
		}
	})
}
