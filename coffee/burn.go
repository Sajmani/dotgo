package main

import (
	"crypto/sha512"
	"log"
	"time"
)

var opsPerSec int = 4000 // for useCPU

func oneOp() {
	var b [sha512.Size]byte
	for j := 0; j < 500; j++ {
		b = sha512.Sum512(b[:])
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
