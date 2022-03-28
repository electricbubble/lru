package lru

import (
	"sync"
	"sync/atomic"
	"testing"
)

func BenchmarkUnsafeLru_Add(b *testing.B) {
	var (
		maxEntries = defaultSize
		counter    uint64
		wg         sync.WaitGroup
		c          = NewUnsafeLru[int, int](maxEntries, WithOnEvictedAsync[int, int](func(k, v int) {
			atomic.AddUint64(&counter, 1)
			wg.Done()
		}))
	)

	delta := b.N - maxEntries
	if delta <= 0 {
		delta = 0
	}
	wg.Add(delta)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Add(i, i)
	}

	wg.Wait()

	b.StopTimer()

	switch {
	case b.N <= maxEntries:
		if counter != 0 {
			b.FailNow()
		}
	default:
		if counter != uint64(b.N-maxEntries) {
			b.Log(counter, b.N)
			b.FailNow()
		}
	}
}

func BenchmarkSafeLru_Add(b *testing.B) {
	var (
		maxEntries = defaultSize
		counter    uint64
		wg         sync.WaitGroup
		c          = New[int, int](maxEntries, WithOnEvictedAsync[int, int](func(k, v int) {
			atomic.AddUint64(&counter, 1)
			wg.Done()
		}))
	)

	delta := b.N
	if delta > maxEntries {
		delta += b.N - maxEntries
	}
	wg.Add(delta)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		go func(i int) {
			defer wg.Done()
			c.Add(i, i)
		}(i)
	}

	wg.Wait()

	b.StopTimer()

	switch {
	case b.N <= maxEntries:
		if counter != 0 {
			b.FailNow()
		}
	default:
		if counter != uint64(b.N-maxEntries) {
			b.FailNow()
		}
	}
}

// go test -bench='Benchmark.+afeLru_Add' . -benchmem
// goos: darwin
// goarch: amd64
// pkg: github.com/electricbubble/lru
// cpu: Intel(R) Core(TM) i5-8259U CPU @ 2.30GHz
// BenchmarkUnsafeLru_Add-8         2793051               430.8 ns/op            81 B/op          3 allocs/op
// BenchmarkSafeLru_Add-8           1059967              1181 ns/op             262 B/op          5 allocs/op
