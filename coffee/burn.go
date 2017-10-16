package main

import (
	"hash/adler32"
	"log"
	"time"
)

// opsPerSec is the baseline for useCPU.
// If zero, it is set automatically during init with a 1-second measurement.
// Set to non-zero to skip that initialization.
// Known values:
// - Sameer's Linux workstation: 54000
var opsPerSec int = 54000

func oneOp() {
	var b [64]byte
	for j := 0; j < 500; j++ {
		b[0] += byte(adler32.Checksum(b[:]))
	}
}

func init() {
	if opsPerSec != 0 {
		return // already set
	}
	var d time.Duration
	var ops int
	for {
		start := time.Now()
		oneOp()
		d += time.Since(start)
		ops++
		if d > 1*time.Second {
			opsPerSec = ops
			log.Print("opsPerSec = ", opsPerSec)
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
