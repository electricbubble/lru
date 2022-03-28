// edited by https://github.com/hashicorp/golang-lru/blob/master/2q.go

package lru

import "sync"

const (
	// default2QRecentRatio is the ratio of the 2Q cache dedicated
	// to recently added entries that have only been accessed once.
	default2QRecentRatio = 0.25

	// default2QGhostEntries is the default ratio of ghost
	// entries kept to track entries recently evicted
	default2QGhostEntries = 0.50
)

// New2Q creates a new TwoQueueCache using the default
// values for the parameters.
func New2Q[K comparable, V any](maxEntries int, opts ...Option[K, V]) *TwoQueueCache[K, V] {
	return New2QParams(maxEntries, default2QRecentRatio, default2QGhostEntries, opts...)
}

func New2QParams[K comparable, V any](maxEntries int, recentRatio, ghostRatio float64, opts ...Option[K, V]) *TwoQueueCache[K, V] {
	if maxEntries <= 0 {
		maxEntries = defaultSize
	}
	if recentRatio < 0.0 || recentRatio > 1.0 {
		recentRatio = default2QRecentRatio
	}
	if ghostRatio < 0.0 || ghostRatio > 1.0 {
		ghostRatio = default2QGhostEntries
	}

	// Determine the sub-sizes
	recentEntries := int(float64(maxEntries) * recentRatio)
	evictEntries := int(float64(maxEntries) * ghostRatio)

	return &TwoQueueCache[K, V]{
		maxEntries:    maxEntries,
		recentEntries: recentEntries,
		recent:        NewUnsafeLru[K, V](maxEntries, opts...),
		frequent:      NewUnsafeLru[K, V](maxEntries, opts...),
		recentEvict:   NewUnsafeLru[K, V](evictEntries, opts...),
	}
}

// TwoQueueCache is a thread-safe fixed size 2Q cache.
// 2Q is an enhancement over the standard LRU cache
// in that it tracks both frequently and recently used
// entries separately. This avoids a burst in access to new
// entries from evicting frequently used entries. It adds some
// additional tracking overhead to the standard LRU cache, and is
// computationally about 2x the cost, and adds some metadata over
// head. The ARCCache is similar, but does not require setting any
// parameters.
type TwoQueueCache[K comparable, V any] struct {
	maxEntries    int
	recentEntries int

	recent      Lru[K, V]
	frequent    Lru[K, V]
	recentEvict Lru[K, V]

	sync.RWMutex
}

// Add a value to the cache.
func (c *TwoQueueCache[K, V]) Add(key K, value V) {
	c.Lock()
	defer c.Unlock()

	// Check if the value is frequently used already,
	// and just update the value
	if c.frequent.Contains(key) {
		c.frequent.Add(key, value)
		return
	}

	// Check if the value is recently used, and promote
	// the value into the frequent list
	if c.recent.Contains(key) {
		c.recent.Remove(key)
		c.frequent.Add(key, value)
		return
	}

	// If the value was recently evicted, add it to the
	// frequently used list
	if c.recentEvict.Contains(key) {
		c.ensureSpace(true)
		c.recentEvict.Remove(key)
		c.frequent.Add(key, value)
		return
	}

	// Add to the recently seen list
	c.ensureSpace(false)
	c.recent.Add(key, value)
}

// Get looks up a key's value from the cache
func (c *TwoQueueCache[K, V]) Get(key K) (value V, ok bool) {
	c.Lock()
	defer c.Unlock()

	// Check if this is a frequent value
	if value, ok = c.frequent.Get(key); ok {
		return
	}

	// If the value is contained in recent, then we
	// promote it to frequent
	if value, ok = c.recent.Peek(key); ok {
		c.recent.Remove(key)
		c.frequent.Add(key, value)
		return
	}

	// No hit
	return value, false
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (c *TwoQueueCache[K, V]) Contains(key K) (ok bool) {
	c.RLock()
	defer c.RUnlock()

	return c.frequent.Contains(key) || c.recent.Contains(key)
}

// Peek is used to inspect the cache value of a key
// without updating recency or frequency.
func (c *TwoQueueCache[K, V]) Peek(key K) (value V, ok bool) {
	c.RLock()
	defer c.RUnlock()

	if value, ok = c.frequent.Peek(key); ok {
		return
	}
	return c.recent.Peek(key)
}

// Remove the provided key from the cache, returning if the key was contained.
func (c *TwoQueueCache[K, V]) Remove(key K) (ok bool) {
	c.Lock()
	defer c.Unlock()

	if c.frequent.Remove(key) {
		return
	}
	if c.recent.Remove(key) {
		return
	}
	if c.recentEvict.Remove(key) {
		return
	}

	return
}

// Keys returns a slice of the keys in the cache.
// The frequently used keys are first in the returned slice.
func (c *TwoQueueCache[K, V]) Keys() []K {
	c.RLock()
	defer c.RUnlock()

	k1 := c.frequent.Keys()
	k2 := c.recent.Keys()
	return append(k1, k2...)
}

// Len returns the number of items in the cache.
func (c *TwoQueueCache[K, V]) Len() int {
	c.RLock()
	defer c.RUnlock()

	return c.recent.Len() + c.frequent.Len()
}

// Clear is used to completely clear the cache
func (c *TwoQueueCache[K, V]) Clear() {
	c.Lock()
	defer c.Unlock()

	c.recent.Clear()
	c.frequent.Clear()
	c.recentEvict.Clear()
}

// ensureSpace is used to ensure we have space in the cache
func (c *TwoQueueCache[K, V]) ensureSpace(recentEvict bool) {
	// If we have space, nothing to do
	recentLen := c.recent.Len()
	freqLen := c.frequent.Len()
	if recentLen+freqLen < c.maxEntries {
		return
	}

	// If the recent buffer is larger than
	// the target, evict from there
	if recentLen > 0 && (recentLen > c.recentEntries || (recentLen == c.recentEntries && !recentEvict)) {
		k, _, _ := c.recent.RemoveOldest()
		var v V
		c.recentEvict.Add(k, v)
		return
	}

	// Remove from the frequent list otherwise
	c.frequent.RemoveOldest()
}
