package grpcconn

import (
	"testing"
	"time"
)

// SetCloseTimeout shortens the Close timeout for the duration of a test.
func SetCloseTimeout(t *testing.T, d time.Duration) {
	old := closeTimeout
	closeTimeout = d
	t.Cleanup(func() { closeTimeout = old })
}
