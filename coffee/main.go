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

var (
	mode      = flag.String("mode", "ideal", "one of ideal, locking, multi-N, finelocking, parsteam, linearpipe-N, splitpipe-N")
	dur       = flag.Duration("dur", 1*time.Second, "perf test duration")
	traceFlag = flag.String("trace", "./trace.out", "execution trace file")
	pars      intList
	maxqs     intList
)

func init() {
	flag.Var(&pars, "par", "perf test parallelism")
	flag.Var(&maxqs, "maxq", "max length of the request queue")
}

// Ideal case: no contention
func idealBrew() {
	useCPU(1 * time.Millisecond)
}

// Locking case: complete contention on a single set of equipent.
var equipment sync.Mutex

func lockingBrew() {
	equipment.Lock()
	grindCoffee()
	makeEspresso()
	steamMilk()
	equipment.Unlock()
}

func grindCoffee() {
	useCPU(333 * time.Microsecond)
}

func makeEspresso() {
	useCPU(334 * time.Microsecond)
}

func steamMilk() {
	useCPU(333 * time.Microsecond)
}

// Fine-grain locking reduces contention.
var grinder, espressoMachine, steamer sync.Mutex

func fineLockingBrew() {
	lockingGrind()
	lockingPress()
	lockingSteam()
}

func lockingGrind() {
	grinder.Lock()
	grindCoffee()
	grinder.Unlock()
}

func lockingPress() {
	espressoMachine.Lock()
	makeEspresso()
	espressoMachine.Unlock()
}

func lockingSteam() {
	steamer.Lock()
	steamMilk()
	steamer.Unlock()
}

// Multiple machines reduce contention.
var grinders, espressoMachines, steamers chan int

func multiBrew() {
	multiGrind()
	multiPress()
	multiSteam()
}

func multiGrind() {
	grinders <- 0
	grindCoffee()
	<-grinders
}

func multiPress() {
	espressoMachines <- 0
	makeEspresso()
	<-espressoMachines
}

func multiSteam() {
	steamers <- 0
	steamMilk()
	<-steamers
}

// Paralellizing operations can help, provided there's available CPU.
// Can steam milk while grinding & pressing, but this loses to
// fine-grain locking when all CPUs utilized.
func parallelSteaming() {
	milk := make(chan int)
	go func() {
		lockingSteam()
		milk <- 0
	}()
	lockingGrind()
	lockingPress()
	<-milk
}

// Linear pipeline
type order struct {
	latte chan int
}
type linearPipeline struct {
	orders, grounds, coffee chan order
	done                    chan int
}

func newLinearPipeline(buffer int) *linearPipeline {
	p := &linearPipeline{
		orders:  make(chan order, buffer),
		grounds: make(chan order, buffer),
		coffee:  make(chan order, buffer),
		done:    make(chan int),
	}
	go p.grinder()
	go p.presser()
	go p.steamer()
	return p
}

func (p *linearPipeline) brew() {
	o := order{make(chan int)}
	p.orders <- o
	<-o.latte
}

func (p *linearPipeline) grinder() {
	for o := range p.orders {
		grindCoffee()
		p.grounds <- o
	}
	close(p.grounds)
}

func (p *linearPipeline) presser() {
	for o := range p.grounds {
		makeEspresso()
		p.coffee <- o
	}
	close(p.coffee)
}

func (p *linearPipeline) steamer() {
	for o := range p.coffee {
		steamMilk()
		o.latte <- 0
	}
	close(p.done)
}

func (p *linearPipeline) close() {
	close(p.orders)
	<-p.done
}

// Split pipeline
type splitOrder struct {
	coffee, milk chan int
}
type splitPipeline struct {
	coffeeOrders, milkOrders, grounds chan splitOrder
	presserDone, steamerDone          chan int
}

func newSplitPipeline(buffer int) *splitPipeline {
	p := &splitPipeline{
		coffeeOrders: make(chan splitOrder, buffer),
		grounds:      make(chan splitOrder, buffer),
		milkOrders:   make(chan splitOrder, buffer),
		presserDone:  make(chan int),
		steamerDone:  make(chan int),
	}
	go p.grinder()
	go p.presser()
	go p.steamer()
	return p
}

func (p *splitPipeline) brew() {
	o := splitOrder{make(chan int, 1), make(chan int, 1)}
	p.coffeeOrders <- o
	p.milkOrders <- o
	<-o.coffee
	<-o.milk
}

func (p *splitPipeline) grinder() {
	for o := range p.coffeeOrders {
		grindCoffee()
		p.grounds <- o
	}
	close(p.grounds)
}

func (p *splitPipeline) presser() {
	for o := range p.grounds {
		makeEspresso()
		o.coffee <- 0
	}
	close(p.presserDone)
}

func (p *splitPipeline) steamer() {
	for o := range p.milkOrders {
		steamMilk()
		o.milk <- 0
	}
	close(p.steamerDone)
}

func (p *splitPipeline) close() {
	close(p.coffeeOrders)
	<-p.presserDone
	close(p.milkOrders)
	<-p.steamerDone
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
	if len(pars) == 0 || len(maxqs) == 0 {
		log.Print("need at least one --par and --maxq")
		os.Exit(1)
	}
	log.Print("mode=", *mode,
		" GOMAXPROCS=", runtime.GOMAXPROCS(0))
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
	var f func()
	var n int
	switch {
	case *mode == "ideal":
		f = idealBrew
	case *mode == "locking":
		f = lockingBrew
	case modeParam(*mode, "multi-", &n):
		grinders = make(chan int, n)
		espressoMachines = make(chan int, n)
		steamers = make(chan int, n)
		f = multiBrew
	case *mode == "finelocking":
		f = fineLockingBrew
	case *mode == "parsteam":
		f = parallelSteaming
	case modeParam(*mode, "linearpipe-", &n):
		p := newLinearPipeline(n)
		defer p.close()
		f = p.brew
	case modeParam(*mode, "splitpipe-", &n):
		p := newSplitPipeline(n)
		defer p.close()
		f = p.brew
	default:
		log.Panicf("unknown mode: %s", *mode)
	}
	fmt.Println(perfResultHeader)
	for _, par := range pars {
		if par == 0 {
			par = runtime.GOMAXPROCS(0)
		}
		for _, maxq := range maxqs {
			res := perfTest(10000, par, maxq, *dur, f)
			fmt.Println(res)
		}
	}
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
