package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

// perfTest runs f repeatedly until d elapses and returns a perfResult
// containing up to maxSamples uniformly selected durations.
func perfTest(maxSamples int, d time.Duration, f func()) (res perfResult) {
	defer res.finish()
	res.samples = make([]time.Duration, 0, maxSamples)
	start := time.Now()
	for {
		now := time.Now()
		if now.Sub(start) >= d {
			break
		}
		f()
		elapsed := time.Since(now)
		res.elapsed += elapsed
		res.ops++
		// Decide whether to include elapsed in samples using
		// https://en.wikipedia.org/wiki/Reservoir_sampling#Algorithm_R
		if len(res.samples) < cap(res.samples) {
			res.samples = append(res.samples, elapsed)
		} else if j := rand.Intn(res.ops); j < len(res.samples) {
			res.samples[j] = elapsed
		}
	}
	return
}

// A perfResult summarizes the results of a perfTest, including
// throughput and a latency distribution.
type perfResult struct {
	ops     int
	elapsed time.Duration
	samples []time.Duration
}

func (res perfResult) opsPerSec() float64 {
	return float64(res.ops) / res.elapsed.Seconds()
}

func (res *perfResult) finish() {
	sort.Slice(res.samples, func(i, j int) bool {
		return res.samples[i] < res.samples[j]
	})
}

func (res perfResult) min() time.Duration  { return res.samples[0] }
func (res perfResult) max() time.Duration  { return res.samples[len(res.samples)-1] }
func (res perfResult) p25() time.Duration  { return res.samples[len(res.samples)/4] }
func (res perfResult) p50() time.Duration  { return res.samples[len(res.samples)/2] }
func (res perfResult) p75() time.Duration  { return res.samples[len(res.samples)*3/4] }
func (res perfResult) p90() time.Duration  { return res.samples[len(res.samples)*9/10] }
func (res perfResult) p99() time.Duration  { return res.samples[len(res.samples)*99/100] }
func (res perfResult) p999() time.Duration { return res.samples[len(res.samples)*999/1000] }

func (res perfResult) String() string {
	return fmt.Sprintf(`throughput: %f ops/sec
min:   %s
p25:   %s
p50:   %s
p75:   %s
p90:   %s
p99:   %s
p999:  %s
max:   %s`,
		res.opsPerSec(), res.min(), res.p25(), res.p50(), res.p75(),
		res.p90(), res.p99(), res.p999(), res.max())
}
