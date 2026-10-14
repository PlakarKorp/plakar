/*
 * Copyright (c) 2026 Gilles Chehade <gilles@poolp.org>
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

// Package secrets flags text files that look like they contain credentials,
// per file and as a count per snapshot.  Only the kind of secret is
// recorded, never what matched.
package secrets

import (
	"bytes"
	"context"
	"regexp"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/analyzer"
)

// headSize bounds what is scanned, and buffered, per file.
const headSize = 1 << 20

var patterns = []struct {
	class string
	re    *regexp.Regexp
}{
	{"secret.private-key", regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH |ENCRYPTED )?PRIVATE KEY-----`)},
	{"secret.aws-access-key", regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"secret.github-token", regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{36,}\b`)},
	{"secret.slack-token", regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}\b`)},
}

func init() {
	if err := analyzer.Register("secrets", New); err != nil {
		panic(err)
	}
}

// Secrets is not safe for concurrent use of Observe and Finalize, which the
// runner never does.
type Secrets struct {
	counts map[string]uint64
}

func New(ctx context.Context, opts *connectors.Options, name string, config map[string]string) (analyzer.Analyzer, error) {
	return &Secrets{counts: make(map[string]uint64)}, nil
}

func (s *Secrets) Name() string    { return "secrets" }
func (s *Secrets) Version() string { return "0.1.0" }

func (s *Secrets) Scope() analyzer.Scope {
	return analyzer.ScopeFile | analyzer.ScopeSnapshot
}

func (s *Secrets) Filter() analyzer.Filter {
	return analyzer.Filter{
		MIMETypes: []string{"text/*", "application/json", "application/xml"},
		Content:   analyzer.ContentHead,
		HeadSize:  headSize,
	}
}

func (s *Secrets) NewProcessor(ctx context.Context, file analyzer.File) (analyzer.Processor, error) {
	return &processor{}, nil
}

func (s *Secrets) Observe(ctx context.Context, pathname string, classes []analyzer.Class) error {
	for _, c := range classes {
		s.counts[c.Name]++
	}
	return nil
}

func (s *Secrets) Finalize(ctx context.Context) ([]analyzer.Class, error) {
	var classes []analyzer.Class
	for _, p := range patterns {
		if n := s.counts[p.class]; n > 0 {
			classes = append(classes, analyzer.Class{Name: p.class, Probability: 1, Count: n})
		}
	}
	return classes, nil
}

func (s *Secrets) Close(ctx context.Context) error {
	return nil
}

// processor buffers the head of the file so that a pattern split across two
// chunks is still found.
type processor struct {
	buf bytes.Buffer
}

func (p *processor) Write(b []byte) (bool, error) {
	p.buf.Write(b)
	return true, nil
}

func (p *processor) Close() ([]analyzer.Class, error) {
	var classes []analyzer.Class
	data := p.buf.Bytes()
	for _, pat := range patterns {
		if pat.re.Match(data) {
			classes = append(classes, analyzer.Class{Name: pat.class, Probability: 1})
		}
	}
	return classes, nil
}
