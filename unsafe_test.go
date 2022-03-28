package lru

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewUnsafeLru(t *testing.T) {
	t.Run("[K string, V int]", func(t *testing.T) {
		c := NewUnsafeLru[string, int](1)
		key := "k"
		val := 1
		c.Add(key, val)
		if v, ok := c.Peek(key); !ok || v != val {
			t.Fatalf("Expected %v, %v, got %v, %v", val, true, v, ok)
		}
	})

	t.Run("[K int, V *struct{}]", func(t *testing.T) {
		type tmpStruct struct {
			val string
		}

		c := NewUnsafeLru[int, *tmpStruct](1)
		key := 1
		val := "v"
		c.Add(key, &tmpStruct{val: val})
		if v, ok := c.Peek(1); !ok || v.val != val {
			t.Fatalf("Expected %v, %v, got %v, %v", val, true, v, ok)
		}
	})

	t.Run("WithOnEvicted", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2)

		evictedData := map[int]int{
			1: 1,
			2: 2,
		}

		c := NewUnsafeLru[int, int](1, WithOnEvicted[int, int](func(key int, value int) {
			v, ok := evictedData[key]
			if !ok || v != value {
				t.Fatalf("evict failed: %v, %v", key, value)
			}
			time.Sleep(time.Second)
			wg.Done()
		}))

		start := time.Now()

		c.Add(1, 1)
		c.Add(2, 2)
		c.Add(3, 3)

		wg.Wait()

		seconds := time.Now().Sub(start).Seconds()
		isExpected := seconds >= 2 && seconds <= 2.1
		if !isExpected {
			t.Fatalf("sync failed")
		}
	})

	t.Run("WithOnEvictedAsync", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(2)

		evictedData := map[int]int{
			1: 1,
			2: 2,
		}

		c := NewUnsafeLru[int, int](1, WithOnEvictedAsync[int, int](func(key int, value int) {
			v, ok := evictedData[key]
			if !ok || v != value {
				t.Fatalf("evict failed: %v, %v", key, value)
			}
			time.Sleep(time.Second)
			wg.Done()
		}))

		start := time.Now()

		c.Add(1, 1)
		c.Add(2, 2)
		c.Add(3, 3)

		wg.Wait()

		seconds := time.Now().Sub(start).Seconds()
		isExpected := seconds >= 1 && seconds <= 1.1
		if !isExpected {
			t.Fatalf("async failed")
		}
	})
}

func Test_unsafeCache_Add(t *testing.T) {
	var (
		i          = 0
		maxEntries = 10
		es         = make([]int, 0, maxEntries)
		c          = NewUnsafeLru[int, int](maxEntries)
	)
	for ; i < maxEntries; i++ {
		c.Add(i, i)
	}

	if c.Len() != maxEntries {
		t.Fatalf("Expected %v, got %v", maxEntries, c.Len())
	}

	for ; i < maxEntries*2; i++ {
		c.Add(i, i)
		es = append(es, i)
	}

	if c.Len() != maxEntries {
		t.Fatalf("Expected %v, got %v", maxEntries, c.Len())
	}

	keys := c.Keys()
	if !reflect.DeepEqual(keys, es) {
		t.Fatalf("keys/values not equal: (%v != %v)", keys, es)
	}
}

func Test_unsafeCache_Get(t *testing.T) {
	var (
		maxEntries = 10
		c          = NewUnsafeLru[int, int](maxEntries)
	)
	for i := 0; i < maxEntries; i++ {
		c.Add(i, i)
	}

	v, ok := c.Get(maxEntries / 2)
	if !ok {
		t.Fatal("should exist")
	}
	if v != maxEntries/2 {
		t.Fatalf("value not equal: (%d != %d)", v, maxEntries/2)
	}

	_, ok = c.Get(maxEntries * maxEntries)
	if ok {
		t.Fatal("should not exist")
	}
}

func Test_unsafeCache_Remove(t *testing.T) {
	var (
		maxEntries = 10
		c          = NewUnsafeLru[int, int](maxEntries)
	)
	for i := 0; i < maxEntries; i++ {
		c.Add(i, i)
	}

	ok := c.Remove(-1)
	if ok {
		t.Fatalf("Expected %v, got %v", !ok, ok)
	}

	ok = c.Remove(0)
	if !ok {
		t.Fatalf("Expected %v, got %v", ok, !ok)
	}
}

func Test_unsafeCache_RemoveOldest(t *testing.T) {
	var (
		maxEntries = 10
		c          = NewUnsafeLru[int, int](maxEntries)
	)

	k, v, ok := c.RemoveOldest()
	if ok {
		t.Fatalf("Expected %v, got %v", !ok, ok)
	}

	for i := 0; i < maxEntries; i++ {
		c.Add(i, i)
	}

	k, v, ok = c.GetOldest()
	if !ok {
		t.Fatalf("Expected %v, got %v", ok, !ok)
	}
	if k != 0 || v != 0 {
		t.Fatalf("Expected %v: %v, got %v: %v", k, v, 0, 0)
	}

	oldK, oldV, oldOK := c.RemoveOldest()
	if !oldOK {
		t.Fatalf("Expected %v, got %v", oldOK, !oldOK)
	}
	if oldK != k || oldV != v {
		t.Fatalf("Expected %v: %v, got %v: %v", k, v, oldK, oldV)
	}
}

func Test_unsafeCache_Clear(t *testing.T) {
	var (
		maxEntries = 10
		c          = NewUnsafeLru[int, int](maxEntries)
	)
	for i := 0; i < maxEntries; i++ {
		c.Add(i, i)
	}

	c.Clear()

	if c.Len() != 0 {
		t.Fatalf("Expected %v, got %v", 0, c.Len())
	}
}

func Test_unsafeCache_Resize(t *testing.T) {
	var (
		maxEntries = 10
		c          = NewUnsafeLru[int, int](maxEntries)
	)
	for i := 0; i < maxEntries; i++ {
		c.Add(i, i)
	}

	evicted := c.Resize(maxEntries / 2)

	if c.Len() != maxEntries/2 {
		t.Fatalf("Expected %v, got %v", maxEntries/2, c.Len())
	}
	if evicted != maxEntries/2 {
		t.Fatalf("Expected %v, got %v", maxEntries/2, evicted)
	}

	evicted = c.Resize(maxEntries * 2)
	if evicted != 0 {
		t.Fatalf("Expected %v, got %v", 0, evicted)
	}

	oldLen := c.Len()
	c.Add(100, 100)
	if c.Len() != oldLen+1 {
		t.Fatalf("Expected %v, got %v", oldLen+1, c.Len())
	}

	size := -1
	diff := c.Len() - size
	evicted = c.Resize(size)
	if evicted != diff {
		t.Fatalf("Expected %v, got %v", diff, evicted)
	}

	c.Add(100, 100)
	c.Add(101, 101)
	if c.Len() != 0 {
		t.Fatalf("Expected %v, got %v", 0, c.Len())
	}
}
