package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"syscall"
	"time"
)

type perfArg struct {
	mode     string        // mode name
	par      int           // request parallelism
	maxq     int           // maximum queue length
	dur      time.Duration // test duration
	interval time.Duration // request interval
}

const perfArgHeader = "mode,par,maxprocs,maxq,dur,interval"

func (arg perfArg) String() string {
	return fmt.Sprintf(
		"%s,%d,%d,%d,%s,%s",
		arg.mode, arg.par, runtime.GOMAXPROCS(0),
		arg.maxq, arg.dur, arg.interval)
}

const maxSamples = 10000

// perfTest runs f repeatedly until arg.dur elapses, then returns a
// perfResult containing up to maxSamples uniformly selected
// durations.  If arg.interval > 0, it specifies the rate at which to
// attempt requests and increment res.drops on failure.
func perfTest(arg perfArg, f func()) (res perfResult) {
	// Pipeline: request generator -> workers -> sampler
	var ru1, ru2 syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &ru1)
	start := time.Now()

	// Generate requests until arg.dur elapses
	stop := time.NewTimer(arg.dur)
	defer stop.Stop()
	var send *time.Ticker
	if arg.interval > 0 {
		send = time.NewTicker(arg.interval)
		defer send.Stop()
	}
	requests := make(chan time.Time, arg.maxq)
	go func() {
		defer close(requests)
		for {
			if send == nil {
				// No request interval: send whenever the queue has space.
				select {
				case <-stop.C:
					return
				case requests <- time.Now():
				}
			} else {
				// Attempt to send a request periodically, drop if queue is full.
				select {
				case <-stop.C:
					return
				case <-send.C:
				}
				select {
				case requests <- time.Now():
				default:
					res.drops++
				}
			}
		}
	}()

	// Workers run f until requests closed.
	durations := make(chan time.Duration)
	var wg sync.WaitGroup
	wg.Add(arg.par)
	for i := 0; i < arg.par; i++ {
		go func() {
			defer wg.Done()
			for start := range requests {
				queueTime := time.Since(start)
				_ = queueTime // not currently used
				start = time.Now()
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
	res.par = arg.par
	defer res.sortSamples()
	res.samples = make([]time.Duration, 0, maxSamples)
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
	par      int
	ops      int
	drops    int // valid iff args.interval > 0
	exectime time.Duration
	walltime time.Duration
	samples  []time.Duration
}

func (res perfResult) opsPerSec() float64 {
	return float64(res.ops) / res.walltime.Seconds()
}

func (res perfResult) dropsPerSec() float64 {
	return float64(res.drops) / res.walltime.Seconds()
}

func (res perfResult) utilization() float64 {
	return res.exectime.Seconds() / res.walltime.Seconds()
}

func (res *perfResult) sortSamples() {
	sort.Slice(res.samples, func(i, j int) bool {
		return res.samples[i] < res.samples[j]
	})
}

func (res perfResult) min() time.Duration { return res.samples[0] }
func (res perfResult) max() time.Duration { return res.samples[len(res.samples)-1] }
func (res perfResult) p05() time.Duration { return res.samples[len(res.samples)/20] }
func (res perfResult) p25() time.Duration { return res.samples[len(res.samples)/4] }
func (res perfResult) p50() time.Duration { return res.samples[len(res.samples)/2] }
func (res perfResult) p75() time.Duration { return res.samples[len(res.samples)*3/4] }
func (res perfResult) p90() time.Duration { return res.samples[len(res.samples)*9/10] }
func (res perfResult) p95() time.Duration { return res.samples[len(res.samples)*95/100] }
func (res perfResult) p99() time.Duration { return res.samples[len(res.samples)*99/100] }

const perfResultHeader = "ops,drops,thru,dthr,wall,exec,util,upar,min,p05,p25,p50,p75,p90,p95,p99,max"

func (res perfResult) String() string {
	return fmt.Sprintf(
		"%d,%d,%.0f,%.0f,%s,%s,%2.f%%,%2.f%%,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f",
		res.ops, res.drops, res.opsPerSec(), res.dropsPerSec(),
		res.walltime, res.exectime, 100*res.utilization(),
		100*res.utilization()/float64(runtime.GOMAXPROCS(0)),
		res.min().Seconds()*1000,
		res.p05().Seconds()*1000, res.p25().Seconds()*1000,
		res.p50().Seconds()*1000, res.p75().Seconds()*1000,
		res.p90().Seconds()*1000, res.p95().Seconds()*1000,
		res.p99().Seconds()*1000, res.max().Seconds()*1000)
}
