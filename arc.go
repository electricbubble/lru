// edited by https://github.com/hashicorp/golang-lru/blob/master/arc.go

package lru

import "sync"

func NewARC[K comparable, V any](maxEntries int, opts ...Option[K, V]) *ARCCache[K, V] {
	if maxEntries <= 0 {
		maxEntries = defaultSize
	}
	return &ARCCache[K, V]{
		maxEntries: maxEntries,
		p:          0,
		t1:         NewUnsafeLru[K, V](maxEntries, opts...),
		b1:         NewUnsafeLru[K, V](maxEntries, opts...),
		t2:         NewUnsafeLru[K, V](maxEntries, opts...),
		b2:         NewUnsafeLru[K, V](maxEntries, opts...),
	}
}

// ARCCache is a thread-safe fixed size Adaptive Replacement Cache (ARC).
// ARC is an enhancement over the standard LRU cache in that tracks both
// frequency and recency of use. This avoids a burst in access to new
// entries from evicting the frequently used older entries. It adds some
// additional tracking overhead to a standard LRU cache, computationally
// it is roughly 2x the cost, and the extra memory overhead is linear
// with the size of the cache. ARC has been patented by IBM, but is
// similar to the TwoQueueCache (2Q) which requires setting parameters.
type ARCCache[K comparable, V any] struct {
	maxEntries int // MaxEntries is the total capacity of the cache
	p          int // P is the dynamic preference towards T1 or T2

	t1 Lru[K, V] // T1 is the LRU for recently accessed items
	b1 Lru[K, V] // B1 is the LRU for evictions from t1

	t2 Lru[K, V] // T2 is the LRU for frequently accessed items
	b2 Lru[K, V] // B2 is the LRU for evictions from t2

	sync.RWMutex
}

// Add a value to the cache
func (c *ARCCache[K, V]) Add(key K, value V) {
	c.Lock()
	defer c.Unlock()

	// Check if the value is contained in T1 (recent), and potentially
	// promote it to frequent T2
	if c.t1.Contains(key) {
		c.t1.Remove(key)
		c.t2.Add(key, value)
		return
	}

	// Check if the value is already in T2 (frequent) and update it
	if c.t2.Contains(key) {
		c.t2.Add(key, value)
		return
	}

	// Check if this value was recently evicted as part of the
	// recently used list
	if c.b1.Contains(key) {
		// T1 set is too small, increase P appropriately
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b2Len > b1Len {
			delta = b2Len / b1Len
		}
		if c.p+delta >= c.maxEntries {
			c.p = c.maxEntries
		} else {
			c.p += delta
		}

		// Potentially need to make room in the cache
		if c.t1.Len()+c.t2.Len() >= c.maxEntries {
			c.replace(false)
		}

		// Remove from B1
		c.b1.Remove(key)

		// Add the key to the frequently used list
		c.t2.Add(key, value)
		return
	}

	// Check if this value was recently evicted as part of the
	// frequently used list
	if c.b2.Contains(key) {
		// T2 set is too small, decrease P appropriately
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b1Len > b2Len {
			delta = b1Len / b2Len
		}
		if delta >= c.p {
			c.p = 0
		} else {
			c.p -= delta
		}

		// Potentially need to make room in the cache
		if c.t1.Len()+c.t2.Len() >= c.maxEntries {
			c.replace(true)
		}

		// Remove from B2
		c.b2.Remove(key)

		// Add the key to the frequently used list
		c.t2.Add(key, value)
		return
	}

	// Potentially need to make room in the cache
	if c.t1.Len()+c.t2.Len() >= c.maxEntries {
		c.replace(false)
	}

	// Keep the size of the ghost buffers trim
	if c.b1.Len() > c.maxEntries-c.p {
		c.b1.RemoveOldest()
	}
	if c.b2.Len() > c.p {
		c.b2.RemoveOldest()
	}

	// Add to the recently seen list
	c.t1.Add(key, value)
}

// Get looks up a key's value from the cache
func (c *ARCCache[K, V]) Get(key K) (value V, ok bool) {
	c.Lock()
	defer c.Unlock()

	// If the value is contained in T1 (recent), then
	// promote it to T2 (frequent)
	if value, ok = c.t1.Peek(key); ok {
		c.t1.Remove(key)
		c.t2.Add(key, value)
		return
	}

	// Check if the value is contained in T2 (frequent)
	if value, ok = c.t2.Get(key); ok {
		return
	}

	// No hit
	return value, false
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (c *ARCCache[K, V]) Contains(key K) (ok bool) {
	c.RLock()
	defer c.RUnlock()

	return c.t1.Contains(key) || c.t2.Contains(key)
}

// Peek is used to inspect the cache value of a key
// without updating recency or frequency.
func (c *ARCCache[K, V]) Peek(key K) (value V, ok bool) {
	c.RLock()
	defer c.RUnlock()

	if value, ok = c.t1.Peek(key); ok {
		return
	}
	return c.t2.Peek(key)
}

// Remove the provided key from the cache, returning if the key was contained.
func (c *ARCCache[K, V]) Remove(key K) (ok bool) {
	c.Lock()
	defer c.Unlock()

	if c.t1.Remove(key) {
		return
	}
	if c.t2.Remove(key) {
		return
	}
	if c.b1.Remove(key) {
		return
	}
	if c.b2.Remove(key) {
		return
	}

	return
}

// Keys returns all the cached keys
func (c *ARCCache[K, V]) Keys() []K {
	c.RLock()
	defer c.RUnlock()

	k1 := c.t1.Keys()
	k2 := c.t2.Keys()
	return append(k1, k2...)
}

// Len returns the number of cached entries
func (c *ARCCache[K, V]) Len() int {
	c.RLock()
	defer c.RUnlock()

	return c.t1.Len() + c.t2.Len()
}

// Clear is used to clear the cache
func (c *ARCCache[K, V]) Clear() {
	c.Lock()
	defer c.Unlock()

	c.t1.Clear()
	c.t2.Clear()
	c.b1.Clear()
	c.b2.Clear()
}

// replace is used to adaptively evict from either T1 or T2
// based on the current learned value of P
func (c *ARCCache[K, V]) replace(b2ContainsKey bool) {
	t1Len := c.t1.Len()
	if t1Len > 0 && (t1Len > c.p || (t1Len == c.p && b2ContainsKey)) {
		k, _, ok := c.t1.RemoveOldest()
		if ok {
			var v V
			c.b1.Add(k, v)
		}
	} else {
		k, _, ok := c.t2.RemoveOldest()
		if ok {
			var v V
			c.b2.Add(k, v)
		}
	}
}
