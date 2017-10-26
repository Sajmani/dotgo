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

type sampler struct {
	min, max time.Duration
	samples  []time.Duration
	count    int
	closed   bool
}

func newSampler() *sampler {
	return &sampler{
		samples: make([]time.Duration, 0, maxSamples),
	}
}

func (s *sampler) checkClosed(want bool) {
	if s.closed != want {
		panic(fmt.Sprintf("sampler: want closed=%v", want))
	}
}

func (s *sampler) add(d time.Duration) {
	s.checkClosed(false)
	if s.count == 0 {
		s.min = d
		s.max = d
	}
	s.count++
	if d < s.min {
		s.min = d
	}
	if d > s.max {
		s.max = d
	}
	// Decide whether to include elapsed in samples using
	// https://en.wikipedia.org/wiki/Reservoir_sampling#Algorithm_R
	if len(s.samples) < cap(s.samples) {
		s.samples = append(s.samples, d)
	} else if j := rand.Intn(s.count); j < len(s.samples) {
		s.samples[j] = d
	}
}

func (s *sampler) close() {
	s.closed = true
	sort.Slice(s.samples, func(i, j int) bool {
		return s.samples[i] < s.samples[j]
	})
}

func (s *sampler) Min() time.Duration {
	s.checkClosed(true)
	return s.min
}

func (s *sampler) Max() time.Duration {
	s.checkClosed(true)
	return s.max
}

func (s *sampler) p(pct int) time.Duration {
	s.checkClosed(true)
	if len(s.samples) == 0 {
		return 0
	}
	return s.samples[len(s.samples)*pct/100]
}

const samplerHeader = "count,min,p05,p25,p50,p75,p90,p95,p99,max"

func (s *sampler) String() string {
	s.checkClosed(true)
	return fmt.Sprintf(
		"%d,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f",
		s.count,
		s.Min().Seconds()*1000,
		s.p(5).Seconds()*1000,
		s.p(25).Seconds()*1000,
		s.p(50).Seconds()*1000,
		s.p(75).Seconds()*1000,
		s.p(90).Seconds()*1000,
		s.p(95).Seconds()*1000,
		s.p(99).Seconds()*1000,
		s.Max().Seconds()*1000)
}

func startUtil() func() (walltime, exectime time.Duration) {
	var ru1, ru2 syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &ru1)
	start := time.Now()
	return func() (walltime, exectime time.Duration) {
		syscall.Getrusage(syscall.RUSAGE_SELF, &ru2)
		return time.Since(start),
			time.Duration(syscall.TimevalToNsec(ru2.Utime) -
				syscall.TimevalToNsec(ru1.Utime))
	}
}

// perfTest runs f repeatedly until arg.dur elapses, then returns a
// perfResult containing up to maxSamples uniformly selected
// durations.  If arg.interval > 0, it specifies the rate at which to
// attempt requests and increment res.drops on failure.
func perfTest(arg perfArg, f func()) (res perfResult) {
	// Pipeline: request generator -> workers -> sampler
	endUtil := startUtil()

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
	res.sampler = newSampler()
	defer res.sampler.close()
	for elapsed := range durations {
		res.sampler.add(elapsed)
	}
	res.walltime, res.exectime = endUtil()
	return
}

// A perfResult summarizes the results of a perfTest, including
// throughput and a latency distribution.
type perfResult struct {
	par      int
	drops    int // valid iff args.interval > 0
	exectime time.Duration
	walltime time.Duration
	sampler  *sampler
}

func (res perfResult) opsPerSec() float64 {
	return float64(res.sampler.count) / res.walltime.Seconds()
}

func (res perfResult) dropsPerSec() float64 {
	return float64(res.drops) / res.walltime.Seconds()
}

func (res perfResult) utilization() float64 {
	return res.exectime.Seconds() / res.walltime.Seconds()
}

const perfResultHeader = "ops,drops,thru,dthr,wall,exec,util,upar,min,p05,p25,p50,p75,p90,p95,p99,max"

func (res perfResult) String() string {
	return fmt.Sprintf(
		"%d,%d,%.0f,%.0f,%s,%s,%2.f%%,%2.f%%,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f,%.3f",
		res.sampler.count,
		res.drops,
		res.opsPerSec(),
		res.dropsPerSec(),
		res.walltime,
		res.exectime,
		100*res.utilization(),
		100*res.utilization()/float64(runtime.GOMAXPROCS(0)),
		res.sampler.Min().Seconds()*1000,
		res.sampler.p(5).Seconds()*1000,
		res.sampler.p(25).Seconds()*1000,
		res.sampler.p(50).Seconds()*1000,
		res.sampler.p(75).Seconds()*1000,
		res.sampler.p(90).Seconds()*1000,
		res.sampler.p(95).Seconds()*1000,
		res.sampler.p(99).Seconds()*1000,
		res.sampler.Max().Seconds()*1000)
}
