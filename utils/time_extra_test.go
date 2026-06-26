package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTimeFlagDateTimeNoSeconds(t *testing.T) {
	// Covers the "2006-01-02 15:04" layout branch.
	expected, _ := time.Parse("2006-01-02 15:04", "2025-04-15 10:30")
	got, err := ParseTimeFlag("2025-04-15 10:30")
	require.NoError(t, err)
	require.Equal(t, expected, got)
}

func TestTimeFlagStringZero(t *testing.T) {
	// nil dest -> empty
	tf := NewTimeFlag(nil)
	require.Equal(t, "", tf.String())

	// zero time -> empty
	var z time.Time
	tf = NewTimeFlag(&z)
	require.Equal(t, "", tf.String())
}

func TestTimeFlagStringNonZero(t *testing.T) {
	tm := time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC)
	tf := NewTimeFlag(&tm)
	require.Equal(t, tm.String(), tf.String())
}

func TestTimeFlagSetValid(t *testing.T) {
	var dest time.Time
	tf := NewTimeFlag(&dest)
	require.NoError(t, tf.Set("2025-04-15"))
	expected, _ := time.Parse("2006-01-02", "2025-04-15")
	require.Equal(t, expected, dest)
	require.Equal(t, expected, *tf.dest)
}

func TestTimeFlagSetInvalid(t *testing.T) {
	var dest time.Time
	tf := NewTimeFlag(&dest)
	err := tf.Set("not-a-time")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid time format")
	// dest left untouched (zero)
	require.True(t, dest.IsZero())
}

func TestTimeFlagSetDuration(t *testing.T) {
	var dest time.Time
	tf := NewTimeFlag(&dest)
	now := time.Now()
	require.NoError(t, tf.Set("1h"))
	require.WithinDuration(t, now.Add(-time.Hour), dest, 2*time.Second)
}
