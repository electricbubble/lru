# Golang-LRU

fixed-size thread safe LRU cache (Go generics).

It is based on [golang/groupcache](https://github.com/golang/groupcache)
`&&` [hashicorp/golang-lru](https://github.com/hashicorp/golang-lru).

## Install

```shell script
go get github.com/electricbubble/lru
```

## Example

```go
package main

import (
	"fmt"
	"github.com/electricbubble/lru"
)

func main() {
	maxEntries := 64
	cache := lru.New[int, any](maxEntries)
	for i := 0; i < maxEntries; i++ {
		cache.Add(i, nil)
	}
	if cache.Len() != maxEntries {
		panic(fmt.Sprintf("bad len: %v", cache.Len()))
	}
}

```

## Benchmark

```shell
go test -bench='Benchmark.+afeLru_Add' . -benchmem
```

```text
goos: darwin
goarch: amd64
pkg: github.com/electricbubble/lru
cpu: Intel(R) Core(TM) i5-8259U CPU @ 2.30GHz
BenchmarkUnsafeLru_Add-8         2793051               430.8 ns/op            81 B/op          3 allocs/op
BenchmarkSafeLru_Add-8           1059967              1181 ns/op             262 B/op          5 allocs/op
```

## Thanks

| |About|
|---|---|
|[golang/groupcache](https://github.com/golang/groupcache)|groupcache is a caching and cache-filling library, intended as a replacement for memcached in many cases.|
|[hashicorp/golang-lru](https://github.com/hashicorp/golang-lru)|Golang LRU cache|

Thank you [JetBrains](https://www.jetbrains.com/?from=gwda) for providing free open source licenses
