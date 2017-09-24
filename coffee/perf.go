package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// perfTest runs f repeatedly in par goroutines until d elapses, then
// returns a perfResult containing up to maxSamples uniformly selected
// durations.
func perfTest(maxSamples int, par int, d time.Duration, f func()) (res perfResult) {
	// Pipeline: request generator -> workers -> sampler
	start := time.Now()

	// Request generator runs until d elapses
	requests := make(chan int)
	go func() {
		defer close(requests)
		for time.Since(start) < d {
			requests <- 0
		}
	}()

	// Workers run f until requests closed.
	durations := make(chan time.Duration)
	var wg sync.WaitGroup
	wg.Add(par)
	for i := 0; i < par; i++ {
		go func() {
			defer wg.Done()
			for _ = range requests {
				start := time.Now()
				f()
				durations <- time.Since(start)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(durations)
	}()

	// Sampler populates result with samples.
	defer res.finish()
	res.samples = make([]time.Duration, 0, maxSamples)
	res.par = par
	for elapsed := range durations {
		res.ops++
		res.exectime += elapsed
		// Decide whether to include elapsed in samples using
		// https://en.wikipedia.org/wiki/Reservoir_sampling#Algorithm_R
		if len(res.samples) < cap(res.samples) {
			res.samples = append(res.samples, elapsed)
		} else if j := rand.Intn(res.ops); j < len(res.samples) {
			res.samples[j] = elapsed
		}
	}
	res.walltime = time.Since(start)
	return
}

// A perfResult summarizes the results of a perfTest, including
// throughput and a latency distribution.
type perfResult struct {
	ops      int
	par      int
	exectime time.Duration
	walltime time.Duration
	samples  []time.Duration
}

func (res perfResult) opsPerSec() float64 {
	return float64(res.ops) / res.walltime.Seconds()
}

func (res perfResult) utilization() float64 {
	return res.exectime.Seconds() / res.walltime.Seconds()
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
	return fmt.Sprintf(`%d ops, %.0f ops/sec, %s walltime, %s exectime, %2.f%% utilization (%2.f%% per thread)
min:   %s
p25:   %s
p50:   %s
p75:   %s
p90:   %s
p99:   %s
p999:  %s
max:   %s`,
		res.ops, res.opsPerSec(), res.walltime, res.exectime,
		100*res.utilization(), 100*res.utilization()/float64(res.par),
		res.min(), res.p25(), res.p50(), res.p75(),
		res.p90(), res.p99(), res.p999(), res.max())
}
