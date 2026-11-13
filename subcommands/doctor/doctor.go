/*
 * Copyright (c) 2026 Plakar contributors
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

package doctor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/exitcodes"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var errMACMismatch = errors.New("blob content does not match its MAC")

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Doctor{} }, 0, "doctor")
}

type Doctor struct {
	subcommands.SubcommandBase

	Samples   int
	Duration  time.Duration
	Threshold float64
	Deep      bool
	Retries   int
}

func (cmd *Doctor) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "doctor [OPTIONS]",
	}
	c.Flags().IntVar(&cmd.Samples, "n", 60000, "maximum number of blobs to read")
	c.Flags().DurationVar(&cmd.Duration, "duration", time.Minute, "wall-clock budget, 0 for none")
	c.Flags().Float64Var(&cmd.Threshold, "threshold", 0, "failure rate in percent above which to exit non-zero")
	c.Flags().BoolVar(&cmd.Deep, "deep", false, "verify the content of each blob against its MAC")
	c.Flags().IntVar(&cmd.Retries, "retries", 2, "times to read a failed blob again to tell transient failures apart")
	return c
}

func (cmd *Doctor) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return fmt.Errorf("too many arguments")
	}
	if cmd.Samples < 1 {
		return fmt.Errorf("-n must be at least 1")
	}
	if cmd.Duration < 0 {
		return fmt.Errorf("-duration must not be negative")
	}
	if cmd.Threshold < 0 || cmd.Threshold > 100 {
		return fmt.Errorf("-threshold must be between 0 and 100")
	}
	if cmd.Retries < 0 {
		return fmt.Errorf("-retries must not be negative")
	}

	cmd.RepositorySecret = ctx.GetSecret()
	return nil
}

type target struct {
	typ      resources.Type
	mac      objects.MAC
	packfile objects.MAC
}

type sample struct {
	target
	elapsed    time.Duration
	size       int
	unverified bool
	err        error

	// recoveredOn is the retry that read a failed blob correctly, 0 if
	// every retry failed too.
	recoveredOn int
}

func (cmd *Doctor) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	start := time.Now()
	targets, total, err := sampleBlobs(ctx, repo, cmd.Samples)
	if err != nil {
		return 1, fmt.Errorf("sample repository state: %w", err)
	}
	if len(targets) == 0 {
		return 1, fmt.Errorf("the repository state has no blobs to sample")
	}

	// -duration bounds the store probes only, so that a large state cannot
	// use up the budget before anything is read.
	var deadline time.Time
	if cmd.Duration > 0 {
		deadline = time.Now().Add(cmd.Duration)
	}

	// Probe with the configured concurrency: the failures this looks for
	// show up under load, not on sequential reads.
	var mu sync.Mutex
	samples := make([]sample, 0, len(targets))
	probing := time.Now()
	wg := new(errgroup.Group)
	wg.SetLimit(ctx.MaxConcurrency)
	for _, t := range targets {
		if ctx.Err() != nil || (!deadline.IsZero() && time.Now().After(deadline)) {
			break
		}
		wg.Go(func() error {
			s := cmd.probe(ctx, repo, t)
			mu.Lock()
			samples = append(samples, s)
			mu.Unlock()
			return nil
		})
	}
	if err := wg.Wait(); err != nil {
		return 1, err
	}
	probed := time.Since(probing)

	if err := ctx.Err(); err != nil {
		return 1, err
	}
	if len(samples) == 0 {
		return 1, fmt.Errorf("no blob was read within -duration %s", cmd.Duration)
	}

	failed, mismatched := report(ctx, samples, total, time.Since(start), probed, cmd.Retries)

	// -threshold tolerates transient read failures, never wrong content.
	if mismatched > 0 {
		return exitcodes.IntegrityFailure, fmt.Errorf("%d of %d blobs do not match their MAC",
			mismatched, len(samples))
	}

	rate := 100 * float64(failed) / float64(len(samples))
	if failed > 0 && rate > cmd.Threshold {
		return exitcodes.Failure, fmt.Errorf("%d of %d blobs failed to read (%.2f%%)",
			failed, len(samples), rate)
	}
	return 0, nil
}

// sampleBlobs draws up to n blobs uniformly from the local state, over all
// resource types, and returns them in random order along with the number of
// blobs the state holds. It never reads from the store.
func sampleBlobs(ctx context.Context, repo *repository.Repository, n int) ([]target, int, error) {
	var picked []target
	seen := 0
	for _, typ := range resources.Types() {
		// Padding blobs are random bytes, not encoded blobs, and cannot be read back.
		if typ == resources.RT_RANDOM {
			continue
		}
		for de, err := range repo.ListBlobs(typ) {
			if err != nil {
				return nil, 0, err
			}
			if err := ctx.Err(); err != nil {
				return nil, 0, err
			}
			seen++
			t := target{typ: de.Type, mac: de.Blob, packfile: de.Location.Packfile}
			if len(picked) < n {
				picked = append(picked, t)
			} else if j := rand.IntN(seen); j < n {
				picked[j] = t
			}
		}
	}
	rand.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
	return picked, seen, nil
}

// hasContentMAC reports whether blobs of this type are keyed by the MAC of
// their decoded content, which is what -deep checks.
func hasContentMAC(typ resources.Type) bool {
	switch typ {
	case resources.RT_CHUNK, resources.RT_OBJECT,
		resources.RT_VFS_ENTRY, resources.RT_VFS_SUMMARY,
		resources.RT_ERROR_ENTRY, resources.RT_XATTR_ENTRY,
		resources.RT_VFS_BTREE, resources.RT_VFS_NODE,
		resources.RT_ERROR_BTREE, resources.RT_ERROR_NODE,
		resources.RT_XATTR_BTREE, resources.RT_XATTR_NODE,
		resources.RT_BTREE_ROOT, resources.RT_BTREE_NODE:
		return true
	}
	return false
}

func (cmd *Doctor) probe(ctx context.Context, repo *repository.Repository, t target) sample {
	s := sample{target: t}

	begin := time.Now()
	s.size, s.unverified, s.err = cmd.read(repo, t)
	s.elapsed = time.Since(begin)

	// A failure is still reported when a retry succeeds: the retry only
	// tells a transient failure from a persistent one.
	if s.err != nil {
		for i := 1; i <= cmd.Retries && ctx.Err() == nil; i++ {
			if _, _, err := cmd.read(repo, t); err == nil {
				s.recoveredOn = i
				break
			}
		}
	}
	return s
}

func (cmd *Doctor) read(repo *repository.Repository, t target) (size int, unverified bool, err error) {
	data, err := repo.GetBlobBytes(t.typ, t.mac)
	if err != nil {
		return 0, false, err
	}
	if cmd.Deep {
		if !hasContentMAC(t.typ) {
			return len(data), true, nil
		}
		if repo.ComputeMAC(data) != t.mac {
			return len(data), false, errMACMismatch
		}
	}
	return len(data), false, nil
}

func report(ctx *appcontext.AppContext, samples []sample, total int, elapsed, probed time.Duration, retries int) (failed, mismatched int) {
	var latencies []time.Duration
	var bytesRead int
	var unverified int
	var transient int

	for _, s := range samples {
		if s.unverified {
			unverified++
		}
		if s.err != nil {
			failed++
			if errors.Is(s.err, errMACMismatch) {
				mismatched++
			}
			retried := ""
			if s.recoveredOn > 0 {
				transient++
				retried = fmt.Sprintf(" (read on retry %d)", s.recoveredOn)
			} else if retries > 0 {
				retried = fmt.Sprintf(" (still failing after %d retries)", retries)
			}
			fmt.Fprintf(ctx.Stdout, "doctor: %s %x (packfile %x): %v%s\n", s.typ, s.mac, s.packfile, s.err, retried)
			continue
		}
		latencies = append(latencies, s.elapsed)
		bytesRead += s.size
	}

	fmt.Fprintf(ctx.Stdout, "doctor: read %d blobs sampled from %d in %s, concurrency %d\n",
		len(samples), total, elapsed.Round(time.Millisecond), ctx.MaxConcurrency)
	fmt.Fprintf(ctx.Stdout, "doctor: failures: %d (%.2f%%)\n",
		failed, 100*float64(failed)/float64(len(samples)))
	if failed > 0 && retries > 0 {
		fmt.Fprintf(ctx.Stdout, "doctor: %d of %d failures were transient: a retry read the blob\n", transient, failed)
	}
	if failed == 0 {
		// A clean sample only bounds the failure rate: say how far, so that a
		// small sample is not taken as proof of health.
		bound := 1 - math.Pow(0.05, 1/float64(len(samples)))
		msg := fmt.Sprintf("doctor: no failure in %d reads: the failure rate is below %.3g%% at 95%% confidence",
			len(samples), 100*bound)
		if oneIn := int(1 / bound); oneIn >= 2 {
			msg += fmt.Sprintf(" (1 in %d)", oneIn)
		}
		fmt.Fprintln(ctx.Stdout, msg)
	}
	if unverified > 0 {
		fmt.Fprintf(ctx.Stdout, "doctor: %d blobs of types without a content MAC were not verified\n", unverified)
	}

	if len(latencies) > 0 {
		slices.Sort(latencies)
		fmt.Fprintf(ctx.Stdout, "doctor: latency: p50=%s p95=%s p99=%s max=%s\n",
			percentile(latencies, 50), percentile(latencies, 95),
			percentile(latencies, 99), latencies[len(latencies)-1])
		if probed > 0 {
			fmt.Fprintf(ctx.Stdout, "doctor: throughput: %.2f MiB/s (%d bytes read)\n",
				float64(bytesRead)/probed.Seconds()/(1<<20), bytesRead)
		}
	}
	return failed, mismatched
}

// percentile returns the nearest-rank percentile of sorted durations.
func percentile(sorted []time.Duration, p float64) time.Duration {
	rank := int(math.Ceil(p / 100 * float64(len(sorted))))
	return sorted[max(rank, 1)-1]
}
