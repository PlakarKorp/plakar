package doctor

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/exitcodes"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestRegisteredFactory(t *testing.T) {
	cmd, _, _ := subcommands.Lookup([]string{"doctor"})
	require.NotNil(t, cmd)
	require.IsType(t, &Doctor{}, cmd)
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"defaults", nil, false},
		{"all flags", []string{"-n", "5", "-deep", "-threshold", "2.5", "-duration", "10s"}, false},
		{"zero samples", []string{"-n", "0"}, true},
		{"negative duration", []string{"-duration", "-1s"}, true},
		{"threshold above 100", []string{"-threshold", "101"}, true},
		{"negative threshold", []string{"-threshold", "-1"}, true},
		{"extra argument", []string{"extra"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
			err := (&Doctor{}).Parse(ctx, tt.args)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// newRepo returns a repository holding one snapshot, and the buffer the
// command writes its report to.
func newRepo(t *testing.T) (*repository.Repository, *appcontext.AppContext, *bytes.Buffer) {
	t.Helper()
	out := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, out, bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
	})
	t.Cleanup(func() { snap.Close() })
	return repo, ctx, out
}

// breakStore removes every packfile from the store while the local state
// still references them, so every blob read fails.
func breakStore(t *testing.T, repo *repository.Repository) {
	t.Helper()
	for mac := range repo.ListPackfiles() {
		require.NoError(t, repo.DeletePackfile(mac))
	}
}

func TestExecute(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		broken     bool
		wantStatus int
		wantErr    bool
		wantOutput []string
	}{
		{
			name:       "healthy store",
			args:       []string{"-n", "20"},
			wantStatus: exitcodes.Success,
			wantOutput: []string{"failures: 0 (0.00%)", "latency: p50=", "throughput:"},
		},
		{
			name:       "sample size is bounded",
			args:       []string{"-n", "3"},
			wantStatus: exitcodes.Success,
			wantOutput: []string{"read 3 blobs sampled from"},
		},
		{
			name:       "deep verifies every blob",
			args:       []string{"-n", "100000", "-deep"},
			wantStatus: exitcodes.Success,
			wantOutput: []string{"failures: 0 (0.00%)"},
		},
		{
			name:       "unreadable store",
			args:       []string{"-n", "10"},
			broken:     true,
			wantStatus: exitcodes.Failure,
			wantErr:    true,
			wantOutput: []string{"failures: 10 (100.00%)", "(packfile "},
		},
		{
			name:       "failures within threshold",
			args:       []string{"-n", "10", "-threshold", "100"},
			broken:     true,
			wantStatus: exitcodes.Success,
			wantOutput: []string{"failures: 10 (100.00%)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, ctx, out := newRepo(t)
			if tt.broken {
				breakStore(t, repo)
			}

			cmd := &Doctor{}
			require.NoError(t, cmd.Parse(ctx, tt.args))

			status, err := cmd.Execute(ctx, repo)
			require.Equal(t, tt.wantStatus, status)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			for _, want := range tt.wantOutput {
				require.Contains(t, out.String(), want)
			}
		})
	}
}

func TestExecuteEmptyRepository(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)

	cmd := &Doctor{}
	require.NoError(t, cmd.Parse(ctx, nil))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, exitcodes.Failure, status)
}
