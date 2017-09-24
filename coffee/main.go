package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/trace"
	"time"
)

var (
	mode = flag.String("mode", "ideal", "one of ideal, ...")
	dur  = flag.Duration("dur", 1*time.Second, "perf test duration")
	par  = flag.Int("par", 1, "perf test parallelism")
)

func idealBarista() {
	idealOrder()
	idealBrew()
	idealServe()
}

func idealOrder() {
	useCPU(1 * time.Millisecond)
}

func idealBrew() {
	useCPU(1 * time.Millisecond)
}

func idealServe() {
	useCPU(1 * time.Millisecond)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if *par == 0 {
		*par = runtime.GOMAXPROCS(0)
	}
	log.Print("mode=", *mode,
		" duration=", *dur,
		" parallelism=", *par,
		" GOMAXPROCS=", runtime.GOMAXPROCS(0))
	traceFile, err := os.Create("./trace.out")
	if err != nil {
		panic(err)
	}
	trace.Start(traceFile)
	defer func() {
		trace.Stop()
		if err := traceFile.Close(); err != nil {
			panic(err)
		}
	}()
	f := idealBarista
	res := perfTest(10000, *par, *dur, f)
	fmt.Println(res)
}
