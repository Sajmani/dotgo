package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime/trace"
	"time"
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
	res := perfTest(1000, 1*time.Second, func() {
		idealBarista()
	})
	fmt.Println(res)
}
