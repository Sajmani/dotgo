package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"syscall"
	"time"
)

// perfTest runs f repeatedly in par goroutines until d elapses, then
// returns a perfResult containing up to maxSamples uniformly selected
// durations.  Maxq specifies the max request queue length.
func perfTest(maxSamples int, par, maxq int, d time.Duration, f func()) (res perfResult) {
	// Pipeline: request generator -> workers -> sampler
	var ru1, ru2 syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &ru1)
	start := time.Now()

	// Request generator runs until d elapses
	requests := make(chan time.Time, maxq)
	go func() {
		defer close(requests)
		for time.Since(start) < d {
			requests <- time.Now()
		}
	}()

	// Workers run f until requests closed.
	durations := make(chan time.Duration)
	var wg sync.WaitGroup
	wg.Add(par)
	for i := 0; i < par; i++ {
		go func() {
			defer wg.Done()
			for start := range requests {
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
	res.maxq = maxq
	for elapsed := range durations {
		res.ops++
		// Decide whether to include elapsed in samples using
		// https://en.wikipedia.org/wiki/Reservoir_sampling#Algorithm_R
		if len(res.samples) < cap(res.samples) {
			res.samples = append(res.samples, elapsed)
		} else if j := rand.Intn(res.ops); j < len(res.samples) {
			res.samples[j] = elapsed
		}
	}
	syscall.Getrusage(syscall.RUSAGE_SELF, &ru2)
	res.exectime = time.Duration(syscall.TimevalToNsec(ru2.Utime) -
		syscall.TimevalToNsec(ru1.Utime))
	res.walltime = time.Since(start)
	return
}

// A perfResult summarizes the results of a perfTest, including
// throughput and a latency distribution.
type perfResult struct {
	ops      int
	par      int
	maxq     int
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

func (res perfResult) min() time.Duration { return res.samples[0] }
func (res perfResult) max() time.Duration { return res.samples[len(res.samples)-1] }
func (res perfResult) p25() time.Duration { return res.samples[len(res.samples)/4] }
func (res perfResult) p50() time.Duration { return res.samples[len(res.samples)/2] }
func (res perfResult) p75() time.Duration { return res.samples[len(res.samples)*3/4] }
func (res perfResult) p90() time.Duration { return res.samples[len(res.samples)*9/10] }
func (res perfResult) p99() time.Duration { return res.samples[len(res.samples)*99/100] }

const perfResultHeader = "par,maxq,ops,thru,wall,exec,util,upar,min,p25,p50,p75,p90,p99,max"

func (res perfResult) String() string {
	return fmt.Sprintf(
		"%d,%d,%d,%.0f,%s,%s,%2.f%%,%2.f%%,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f",
		res.par, res.maxq, res.ops, res.opsPerSec(), res.walltime, res.exectime,
		100*res.utilization(), 100*res.utilization()/float64(res.par),
		res.min().Seconds()*1000, res.p25().Seconds()*1000,
		res.p50().Seconds()*1000, res.p75().Seconds()*1000,
		res.p90().Seconds()*1000, res.p99().Seconds()*1000,
		res.max().Seconds()*1000)
}
