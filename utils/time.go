/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package utils

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/go-human2duration"
)

// TimeFlag implements flag.Value interface
type TimeFlag struct {
	dest *time.Time
}

func NewTimeFlag(dest *time.Time) *TimeFlag {
	return &TimeFlag{dest}
}

func (t *TimeFlag) String() string {
	if t.dest == nil || t.dest.IsZero() {
		return ""
	}
	return t.dest.String()
}

func (t *TimeFlag) Set(s string) error {
	parsed, err := ParseTimeFlag(s)
	if err != nil {
		return err
	}
	*t.dest = parsed
	return nil
}

func ParseTimeFlag(input string) (time.Time, error) {
	if input == "" {
		return time.Time{}, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, input); err == nil {
			return t, nil
		}
	}

	// If none of the date layouts match, try to parse it as a duration.
	d, err := human2duration.ParseDuration(input)
	if err == nil {
		return time.Now().Add(-d), nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %q", input)
}
