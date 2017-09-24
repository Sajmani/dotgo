package main

import (
	"log"
	"time"
)

func main() {
	shop := NewCoffeeShop()
	start := time.Now()
	shop.Latte()
	log.Print(time.Since(start))
}

func NewCoffeeShop() *CoffeeShop {
	return &CoffeeShop{
		tills:    newIntChan(1),
		grinders: newIntChan(1),
		brewers:  newIntChan(1),
		steamers: newIntChan(1),
	}
}

func newIntChan(n int) chan int {
	c := make(chan int, n)
	for i := 0; i < n; i++ {
		c <- 0
	}
	return c
}

type CoffeeShop struct {
	tills    chan int
	grinders chan int
	brewers  chan int
	steamers chan int
}

func (s *CoffeeShop) Latte() {
	s.Order()
	s.Grind()
	s.Brew()
	s.Steam()
	s.Assemble()
}

func (s *CoffeeShop) Order() {
	n := <-s.tills
	n++
	useCPU(5 * time.Millisecond)
	s.tills <- n
}

func (s *CoffeeShop) Grind() {
	n := <-s.grinders
	n++
	useCPU(5 * time.Millisecond)
	s.grinders <- n
}

func (s *CoffeeShop) Brew() {
	n := <-s.brewers
	n++
	useCPU(5 * time.Millisecond)
	s.brewers <- n
}

func (s *CoffeeShop) Steam() {
	n := <-s.steamers
	n++
	useCPU(5 * time.Millisecond)
	s.steamers <- n
}

func (s *CoffeeShop) Assemble() {
	useCPU(5 * time.Millisecond)
}
