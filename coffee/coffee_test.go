package main

import (
	"testing"
)

func BenchmarkCoffee(b *testing.B) {
	shop := NewCoffeeShop()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			shop.Latte()
			b.SetBytes(1e6)
		}
	})
}
