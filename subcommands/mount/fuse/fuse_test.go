//go:build linux || darwin

package fuse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anacrolix/fuse"
	"github.com/stretchr/testify/require"
)

func noRelease(t *testing.T) func(*fuse.Conn) {
	return func(*fuse.Conn) { t.Error("release must not be called") }
}

func TestMountCancellable_ReturnsConn(t *testing.T) {
	want := &fuse.Conn{}
	got, err := mountCancellable(context.Background(),
		func() (*fuse.Conn, error) { return want, nil },
		noRelease(t))
	require.NoError(t, err)
	require.Same(t, want, got)
}

func TestMountCancellable_ReturnsMountError(t *testing.T) {
	mountErr := errors.New("cannot locate OSXFUSE")
	got, err := mountCancellable(context.Background(),
		func() (*fuse.Conn, error) { return nil, mountErr },
		noRelease(t))
	require.ErrorIs(t, err, mountErr)
	require.Nil(t, got)
}

// The B5 hang: mount never returns, Ctrl+C must still get us out.
func TestMountCancellable_CancelWhileMountHangs(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	hang := make(chan struct{})
	defer close(hang) // let the fake mount goroutine finish

	interrupted := errors.New("interrupted")
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel(interrupted)
	}()

	done := make(chan error, 1)
	go func() {
		_, err := mountCancellable(ctx,
			func() (*fuse.Conn, error) { <-hang; return nil, errors.New("unblocked") },
			noRelease(t))
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, interrupted)
	case <-time.After(5 * time.Second):
		t.Fatal("mountCancellable did not return after cancellation")
	}
}

// A mount that succeeds after the caller gave up must be released, not leaked.
func TestMountCancellable_LateMountIsReleased(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	proceed := make(chan struct{})
	late := &fuse.Conn{}
	released := make(chan *fuse.Conn, 1)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := mountCancellable(ctx,
		func() (*fuse.Conn, error) { <-proceed; return late, nil },
		func(c *fuse.Conn) { released <- c })
	require.ErrorIs(t, err, context.Canceled)

	close(proceed) // the mount now "succeeds", after the caller returned
	select {
	case c := <-released:
		require.Same(t, late, c)
	case <-time.After(5 * time.Second):
		t.Fatal("late mount was not released")
	}
}
