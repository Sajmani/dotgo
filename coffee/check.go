package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

func checkVariance() {
	par := 1
	if len(pars) > 0 {
		par = pars[0]
	}

	timings := utilization()
	var wg sync.WaitGroup
	wg.Add(par)
	for i := 0; i < par; i++ {
		go func() {
			checkFunc("no-locks", func() {
				useCPU(*duration)
			})
			wg.Done()
		}()
	}
	wg.Wait()
	wall, exec := timings()
	log.Println("no-locks utilization", exec.Seconds()/wall.Seconds())

	timings = utilization()
	var mu sync.Mutex
	wg.Add(par)
	for i := 0; i < par; i++ {
		go func() {
			checkFunc("one-lock", func() {
				mu.Lock()
				useCPU(*duration)
				mu.Unlock()
			})
			wg.Done()
		}()
	}
	wg.Wait()
	wall, exec = timings()
	log.Println("one-lock utilization", exec.Seconds()/wall.Seconds())

	timings = utilization()
	var mu1, mu2, mu3 sync.Mutex
	wg.Add(par)
	for i := 0; i < par; i++ {
		go func() {
			checkFunc("three-locks", func() {
				mu1.Lock()
				useCPU(*duration)
				mu1.Unlock()

				mu2.Lock()
				useCPU(*duration)
				mu2.Unlock()

				mu3.Lock()
				useCPU(*duration)
				mu3.Unlock()
			})
			wg.Done()
		}()
	}
	wg.Wait()
	wall, exec = timings()
	log.Println("three-locks utilization", exec.Seconds()/wall.Seconds())
}

func checkFunc(kind string, f func()) {
	var ds []time.Duration
	for i := 0; i < 60; i++ {
		start := time.Now()
		f()
		elapsed := time.Since(start)
		ds = append(ds, elapsed)
	}
	log.Println(kind, millis(ds))
}

func millis(ds []time.Duration) string {
	var s string
	for i := 0; i < len(ds); i++ {
		s += fmt.Sprintf("%.0d ", ds[i].Nanoseconds()/1e6)
	}
	return s
}
