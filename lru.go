package lru

import "sync"

const defaultSize = 128

type Lru[K comparable, V any] interface {
	// Add a value to the cache. Returns true if an eviction occurred.
	Add(key K, value V) (evicted bool)

	// Get looks up a key's value from the cache
	Get(key K) (value V, ok bool)

	// Contains checks if a key is in the cache, without updating the recent-ness
	// or deleting it for being stale.
	Contains(key K) (ok bool)

	// Peek returns the key value (or undefined if not found) without updating
	// the "recently used"-ness of the key.
	Peek(key K) (value V, ok bool)

	// Remove removes the provided key from the cache, returning if the
	// key was contained.
	Remove(key K) (ok bool)

	// RemoveOldest removes the oldest item from the cache.
	RemoveOldest() (key K, value V, ok bool)

	// GetOldest returns the oldest entry
	GetOldest() (key K, value V, ok bool)

	// Keys returns a slice of the keys in the cache, from oldest to newest.
	Keys() []K

	// Len returns the number of items in the cache.
	Len() int

	// Resize changes the cache size.
	Resize(size int) (evicted int)

	// Clear is used to completely clear the cache
	Clear()
}

func New[K comparable, V any](maxEntries int, opts ...Option[K, V]) *Cache[K, V] {
	return &Cache[K, V]{
		lru: NewUnsafeLru[K, V](maxEntries, opts...),
	}
}

var _ Lru[int, int] = (*Cache[int, int])(nil)

// Cache is an LRU cache. It is safe for concurrent access.
type Cache[K comparable, V any] struct {
	lru Lru[K, V]

	sync.RWMutex
}

// Add a value to the cache. Returns true if an eviction occurred.
func (c *Cache[K, V]) Add(key K, value V) (evicted bool) {
	c.Lock()
	defer c.Unlock()

	return c.lru.Add(key, value)
}

// Get looks up a key's value from the cache
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.Lock()
	defer c.Unlock()

	return c.lru.Get(key)
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *Cache[K, V]) Contains(key K) (ok bool) {
	c.RLock()
	defer c.RUnlock()

	return c.lru.Contains(key)
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *Cache[K, V]) Peek(key K) (value V, ok bool) {
	c.RLock()
	defer c.RUnlock()

	return c.lru.Peek(key)
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *Cache[K, V]) Remove(key K) (ok bool) {
	c.Lock()
	defer c.Unlock()

	return c.lru.Remove(key)
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache[K, V]) RemoveOldest() (key K, value V, ok bool) {
	c.Lock()
	defer c.Unlock()

	return c.lru.RemoveOldest()
}

// GetOldest returns the oldest entry
func (c *Cache[K, V]) GetOldest() (key K, value V, ok bool) {
	c.RLock()
	defer c.RUnlock()

	return c.lru.GetOldest()
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *Cache[K, V]) Keys() []K {
	c.RLock()
	defer c.RUnlock()

	return c.lru.Keys()
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int {
	c.RLock()
	defer c.RUnlock()

	return c.lru.Len()
}

// Resize changes the cache size.
func (c *Cache[K, V]) Resize(size int) (evicted int) {
	c.Lock()
	defer c.Unlock()

	return c.lru.Resize(size)
}

// Clear is used to completely clear the cache
func (c *Cache[K, V]) Clear() {
	c.Lock()
	defer c.Unlock()

	c.lru.Clear()
}
