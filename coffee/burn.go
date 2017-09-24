package main

import (
	"crypto/sha512"
	"time"
)

var opsPerSec int // for useCPU

func oneOp() {
	var b [sha512.Size]byte
	for j := 0; j < 500; j++ {
		b = sha512.Sum512(b[:])
	}
}

func init() {
	var d time.Duration
	var ops int
	for {
		start := time.Now()
		oneOp()
		d += time.Since(start)
		ops++
		if d > 100*time.Millisecond {
			opsPerSec = ops * 10
			return
		}
	}
}

// useCPU uses specified cpu-seconds.
func useCPU(d time.Duration) {
	c := opsPerSec * int(d) / int(time.Second)
	for i := 0; i < c; i++ {
		oneOp()
	}
}
