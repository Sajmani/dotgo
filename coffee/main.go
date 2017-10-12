// The coffee command simulates a small parallel pipeline and outputs CSV.
//
// The pipeline consists of three stages: grinding coffee beans,
// preparing espresso, and steaming milk.  Each stage contends on the
// respective machine (grinder, espresso machine, steamer).
//
// This simulation reports throughput, latency, and utilization.
// It can also create an execution trace with the --trace flag.
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	mode = flag.String("mode", "ideal", `comma-separated list of modes:
ideal: no synchronization, no contention overhead.  Fails the race detector.
locking: one lock, maximal contention.
finelocking: one lock per machine, permitting greater parallelism.
parsteam: finelocking with steaming happening in parallel with the other stages.
linearpipe-N: a pipeline with one goroutine per machine.
splitpipe-N: a pipeline with the steamer stage happening in parallel with the other stages.
multi-N: finelocking but with N copies of each machine.
multipipe-N: N copies of linearpipe.
`)
	duration  = flag.Duration("dur", 1*time.Second, "perf test duration")
	interval  = flag.Duration("interval", 0, "perf test request interval")
	traceFlag = flag.String("trace", "", "execution trace file, e.g., ./trace.out")
	pars      intList
	maxqs     intList
)

func init() {
	flag.Var(&pars, "par", "comma-separated list of perf test parallelism (how many brews to run in parallel)")
	flag.Var(&maxqs, "maxq", "comma-separated max lengths of the request queue (how many calls to queue up)")
}

type intList []int

func (il *intList) Set(s string) error {
	ss := strings.Split(s, ",")
	for _, s := range ss {
		n, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*il = append(*il, n)
	}
	return nil
}

func (il *intList) String() string {
	var ss []string
	for _, n := range *il {
		ss = append(ss, strconv.Itoa(n))
	}
	return strings.Join(ss, ",")
}

// Shared state, requiring synchronization
var grindCount, pressCount, steamCount int

// Named types for the pipeline elements.
type (
	grounds int
	coffee  int
	milk    int
	latte   int
)

// Ideal case: no contention (fails the race detector with par > 1)
func idealBrew() latte {
	grounds := grindCoffee(&grindCount)
	coffee := makeEspresso(&pressCount, grounds)
	milk := steamMilk(&steamCount)
	return makeLatte(coffee, milk)
}

// Simulate one millisecond of prep time.

func grindCoffee(count *int) grounds {
	*count++
	useCPU(250 * time.Microsecond)
	return grounds(0)
}

func makeEspresso(count *int, grounds grounds) coffee {
	*count++
	useCPU(250 * time.Microsecond)
	return coffee(grounds)
}

func steamMilk(count *int) milk {
	*count++
	useCPU(250 * time.Microsecond)
	return milk(0)
}

func makeLatte(coffee coffee, milk milk) latte {
	// No shared state to contend on.
	useCPU(250 * time.Microsecond)
	return latte(int(coffee) + int(milk))
}

// Locking case: complete contention on a single set of equipent.
var equipment sync.Mutex

func lockingBrew() latte {
	equipment.Lock()
	defer equipment.Unlock()
	grounds := grindCoffee(&grindCount)
	coffee := makeEspresso(&pressCount, grounds)
	milk := steamMilk(&steamCount)
	return makeLatte(coffee, milk)
}

// Fine-grain locking reduces contention.
var grinder, espressoMachine, steamer sync.Mutex

func fineLockingBrew() latte {
	grounds := lockingGrind()
	coffee := lockingPress(grounds)
	milk := lockingSteam()
	return makeLatte(coffee, milk)
}

func lockingGrind() grounds {
	grinder.Lock()
	defer grinder.Unlock()
	return grindCoffee(&grindCount)
}

func lockingPress(grounds grounds) coffee {
	espressoMachine.Lock()
	defer espressoMachine.Unlock()
	return makeEspresso(&pressCount, grounds)
}

func lockingSteam() milk {
	steamer.Lock()
	defer steamer.Unlock()
	return steamMilk(&steamCount)
}

// Paralellizing operations can help, provided there's available CPU.
// Can steam milk while grinding & pressing, but this loses to
// fine-grain locking when all CPUs utilized.
func parallelSteaming() latte {
	c := make(chan milk)
	go func() {
		c <- lockingSteam()
	}()
	grounds := lockingGrind()
	coffee := lockingPress(grounds)
	milk := <-c
	return makeLatte(coffee, milk)
}

// Multiple machines reduce contention.
var grinders, espressoMachines, steamers chan int

// newZeroes returns a channel containing n zeroes in its buffer.
func newZeroes(n int) chan int {
	c := make(chan int, n)
	for i := 0; i < n; i++ {
		c <- 0
	}
	return c
}

func multiBrew() latte {
	grounds := multiGrind()
	coffee := multiPress(grounds)
	milk := multiSteam()
	return makeLatte(coffee, milk)
}

func multiGrind() grounds {
	count := <-grinders
	grounds := grindCoffee(&count)
	grinders <- count
	return grounds
}

func multiPress(grounds grounds) coffee {
	count := <-espressoMachines
	coffee := makeEspresso(&count, grounds)
	espressoMachines <- count
	return coffee
}

func multiSteam() milk {
	count := <-steamers
	milk := steamMilk(&count)
	steamers <- count
	return milk
}

// Linear pipeline
type order struct {
	grounds grounds
	coffee  coffee
	milk    chan milk
}

type linearPipeline struct {
	orders            chan order
	ordersWithGrounds chan order
	ordersWithCoffee  chan order
	done              chan int
}

func newLinearPipeline(buffer int) *linearPipeline {
	p := &linearPipeline{
		orders:            make(chan order, buffer),
		ordersWithGrounds: make(chan order, buffer),
		ordersWithCoffee:  make(chan order, buffer),
		done:              make(chan int),
	}
	go p.grinder()
	go p.presser()
	go p.steamer()
	return p
}

func (p *linearPipeline) brew() latte {
	o := order{milk: make(chan milk, 1)}
	p.orders <- o
	milk := <-o.milk
	return makeLatte(o.coffee, milk)
}

func (p *linearPipeline) grinder() {
	for o := range p.orders {
		o.grounds = grindCoffee(&grindCount)
		p.ordersWithGrounds <- o
	}
	close(p.ordersWithGrounds)
}

func (p *linearPipeline) presser() {
	for o := range p.ordersWithGrounds {
		o.coffee = makeEspresso(&pressCount, o.grounds)
		p.ordersWithCoffee <- o
	}
	close(p.ordersWithCoffee)
}

func (p *linearPipeline) steamer() {
	for o := range p.ordersWithCoffee {
		o.milk <- steamMilk(&steamCount)
	}
	close(p.done)
}

func (p *linearPipeline) close() {
	close(p.orders)
	<-p.done
}

// Split pipeline
type splitOrder struct {
	grounds grounds
	coffee  chan coffee
	milk    chan milk
}
type splitPipeline struct {
	coffeeOrders, milkOrders, ordersWithGrounds chan splitOrder
	presserDone, steamerDone                    chan int
}

func newSplitPipeline(buffer int) *splitPipeline {
	p := &splitPipeline{
		coffeeOrders:      make(chan splitOrder, buffer),
		ordersWithGrounds: make(chan splitOrder, buffer),
		milkOrders:        make(chan splitOrder, buffer),
		presserDone:       make(chan int),
		steamerDone:       make(chan int),
	}
	go p.grinder()
	go p.presser()
	go p.steamer()
	return p
}

func (p *splitPipeline) brew() latte {
	o := splitOrder{
		coffee: make(chan coffee, 1),
		milk:   make(chan milk, 1),
	}
	p.coffeeOrders <- o
	p.milkOrders <- o
	milk := <-o.milk // receive in reverse order of send to avoid deadlock
	coffee := <-o.coffee
	return makeLatte(coffee, milk)
}

func (p *splitPipeline) grinder() {
	for o := range p.coffeeOrders {
		o.grounds = grindCoffee(&grindCount)
		p.ordersWithGrounds <- o
	}
	close(p.ordersWithGrounds)
}

func (p *splitPipeline) presser() {
	for o := range p.ordersWithGrounds {
		o.coffee <- makeEspresso(&pressCount, o.grounds)
	}
	close(p.presserDone)
}

func (p *splitPipeline) steamer() {
	for o := range p.milkOrders {
		o.milk <- steamMilk(&steamCount)
	}
	close(p.steamerDone)
}

func (p *splitPipeline) close() {
	close(p.coffeeOrders)
	<-p.presserDone
	close(p.milkOrders)
	<-p.steamerDone
}

// Multiple copies of linearPipeline, like multiple coffee shops.
type multiPipeline struct {
	pipes chan *linearPipeline
}

func newMultiPipeline(n int) *multiPipeline {
	p := &multiPipeline{
		pipes: make(chan *linearPipeline, n),
	}
	for i := 0; i < n; i++ {
		p.pipes <- newLinearPipeline(0) // no buffering
	}
	return p
}

func (p *multiPipeline) brew() latte {
	lp := <-p.pipes
	o := order{milk: make(chan milk, 1)}
	lp.orders <- o
	p.pipes <- lp    // release the pipeline for other brew calls
	milk := <-o.milk // THEN wait for order to complete
	return makeLatte(o.coffee, milk)
}

func (p *multiPipeline) close() {
	close(p.pipes)
	for lp := range p.pipes {
		lp.close()
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	log.Print("GOMAXPROCS=", runtime.GOMAXPROCS(0))
	flag.Parse()
	if len(pars) == 0 {
		pars = []int{1}
	}
	if len(maxqs) == 0 {
		maxqs = []int{0}
	}
	modes := strings.Split(*mode, ",")
	if len(modes) == 0 {
		modes = []string{"ideal"}
	}
	if *traceFlag != "" {
		traceFile, err := os.Create(*traceFlag)
		if err != nil {
			panic(err)
		}
		trace.Start(traceFile)
		defer func() {
			trace.Stop()
			if err := traceFile.Close(); err != nil {
				log.Panic(err)
			}
		}()
	}
	// Run all combinations of modes, parallelisms, and maxqs.
	// Print output as CSV.
	fmt.Println(perfArgHeader + "," + perfResultHeader)
	for _, mode := range modes {
		f, close := modeFunc(mode)
		for _, par := range pars {
			if par == 0 {
				par = runtime.GOMAXPROCS(0)
			}
			for _, maxq := range maxqs {
				arg := perfArg{
					mode:     mode,
					par:      par,
					maxq:     maxq,
					dur:      *duration,
					interval: *interval,
				}
				res := perfTest(arg, func() { f() })
				fmt.Println(arg.String() + "," + res.String())
			}
		}
		if close != nil {
			close()
		}
	}
}

func modeFunc(mode string) (func() latte, func()) {
	var n int
	switch {
	case mode == "ideal":
		return idealBrew, nil
	case mode == "locking":
		return lockingBrew, nil
	case modeParam(mode, "multi-", &n):
		grinders = newZeroes(n)
		espressoMachines = newZeroes(n)
		steamers = newZeroes(n)
		return multiBrew, func() {
			grinders, espressoMachines, steamers = nil, nil, nil
		}
	case mode == "finelocking":
		return fineLockingBrew, nil
	case mode == "parsteam":
		return parallelSteaming, nil
	case modeParam(mode, "linearpipe-", &n):
		p := newLinearPipeline(n)
		return p.brew, p.close
	case modeParam(mode, "splitpipe-", &n):
		p := newSplitPipeline(n)
		return p.brew, p.close
	case modeParam(mode, "multipipe-", &n):
		p := newMultiPipeline(n)
		return p.brew, p.close
	}
	log.Panicf("unknown mode: %s", mode)
	return nil, nil
}

func modeParam(mode, prefix string, n *int) bool {
	if !strings.HasPrefix(mode, prefix) {
		return false
	}
	var err error
	*n, err = strconv.Atoi((mode)[len(prefix):])
	if err != nil {
		log.Panicf("bad mode %s: %v", mode, err)
	}
	return true
}
