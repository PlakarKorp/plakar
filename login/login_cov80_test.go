package login

import (
	"strings"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
)

// Poll with zero iterations never enters the loop (so it never touches the
// network) and returns the "could not obtain token" sentinel.
func TestLoginCov80PollZeroIterations(t *testing.T) {
	t.Parallel()
	ctx := appcontext.NewAppContext()
	flow, _ := NewLoginFlow(ctx, true)

	progressCalled := false
	token, err := flow.Poll("poll-id", 0, time.Second, func() { progressCalled = true })
	if token != "" {
		t.Fatalf("token = %q, want empty", token)
	}
	if err == nil {
		t.Fatal("expected error after exhausting iterations, got nil")
	}
	if !strings.Contains(err.Error(), "could not obtain token after 0 iterations") {
		t.Fatalf("err = %v, want exhausted-iterations sentinel", err)
	}
	if progressCalled {
		t.Fatal("progress callback should not fire with zero iterations")
	}
}

// Run rejects an unknown provider before doing any network I/O. (Distinct
// from the existing twitter case to also lock in the empty-string provider.)
func TestLoginCov80RunEmptyProvider(t *testing.T) {
	t.Parallel()
	ctx := appcontext.NewAppContext()
	flow, _ := NewLoginFlow(ctx, true)
	_, err := flow.Run("", map[string]string{"k": "v"})
	if err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("Run(\"\") err = %v, want unsupported provider", err)
	}
}

// RunUI likewise rejects an unknown provider before any network I/O.
func TestLoginCov80RunUIEmptyProvider(t *testing.T) {
	t.Parallel()
	ctx := appcontext.NewAppContext()
	flow, _ := NewLoginFlow(ctx, false)
	_, err := flow.RunUI("nope", map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "unsupported provider") {
		t.Fatalf("RunUI err = %v, want unsupported provider", err)
	}
}
