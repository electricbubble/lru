package lru

import "github.com/electricbubble/lru/list"

type Option[K comparable, V any] func(*unsafeCache[K, V])

func WithOnEvicted[K comparable, V any](onEvicted func(key K, value V)) Option[K, V] {
	return func(c *unsafeCache[K, V]) {
		c.onEvicted = onEvicted
		c.async = false
	}
}

func WithOnEvictedAsync[K comparable, V any](onEvicted func(key K, value V)) Option[K, V] {
	return func(c *unsafeCache[K, V]) {
		c.onEvicted = onEvicted
		c.async = true
	}
}

func NewUnsafeLru[K comparable, V any](maxEntries int, opts ...Option[K, V]) Lru[K, V] {
	if maxEntries <= 0 {
		maxEntries = defaultSize
	}
	c := &unsafeCache[K, V]{
		maxEntries: maxEntries,
		entries:    list.New[*entry[K, V]](),
		bucket:     make(map[K]*list.Element[*entry[K, V]]),
	}
	for _, fn := range opts {
		if fn == nil {
			continue
		}
		fn(c)
	}
	return c
}

var _ Lru[int, any] = (*unsafeCache[int, any])(nil)

// unsafeCache is an LRU cache. It is not safe for concurrent access.
type unsafeCache[K comparable, V any] struct {
	// maxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	maxEntries int

	// onEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	onEvicted func(key K, value V)
	async     bool

	entries *list.List[*entry[K, V]]
	bucket  map[K]*list.Element[*entry[K, V]]
}

// entry is used to hold a value in the entries
type entry[K comparable, V any] struct {
	key   K
	value V
}

func (c *unsafeCache[K, V]) Add(key K, value V) (evicted bool) {
	// Check for existing item
	if elem, ok := c.bucket[key]; ok {
		c.entries.MoveToFront(elem)
		elem.Value.value = value
		return false
	}

	// Add new item
	ent := &entry[K, V]{key, value}
	elem := c.entries.PushFront(ent)
	c.bucket[key] = elem

	evicted = c.entries.Len() > c.maxEntries
	// Verify size not exceeded
	if evicted {
		c.removeOldest()
	}
	return evicted
}

func (c *unsafeCache[K, V]) Get(key K) (value V, ok bool) {
	var elem *list.Element[*entry[K, V]]
	if elem, ok = c.bucket[key]; !ok {
		return
	}

	c.entries.MoveToFront(elem)
	if elem.Value == nil {
		return value, false
	}

	value = elem.Value.value
	return
}

func (c *unsafeCache[K, V]) Contains(key K) (ok bool) {
	_, ok = c.bucket[key]
	return ok
}

func (c *unsafeCache[K, V]) Peek(key K) (value V, ok bool) {
	var elem *list.Element[*entry[K, V]]
	if elem, ok = c.bucket[key]; !ok {
		return
	}

	value = elem.Value.value
	return
}

func (c *unsafeCache[K, V]) Remove(key K) (ok bool) {
	var elem *list.Element[*entry[K, V]]
	if elem, ok = c.bucket[key]; !ok {
		return
	}

	c.removeElement(elem)
	return
}

func (c *unsafeCache[K, V]) RemoveOldest() (key K, value V, ok bool) {
	elem := c.entries.Back()
	if elem == nil {
		return key, value, false
	}

	c.removeElement(elem)
	ent := elem.Value
	key = ent.key
	value = ent.value
	return key, value, true
}

func (c *unsafeCache[K, V]) GetOldest() (key K, value V, ok bool) {
	elem := c.entries.Back()
	if elem == nil {
		return key, value, false
	}

	ent := elem.Value
	key = ent.key
	value = ent.value
	return key, value, true
}

func (c *unsafeCache[K, V]) Keys() []K {
	keys := make([]K, c.entries.Len())
	for i, elem := 0, c.entries.Back(); elem != nil; i, elem = i+1, elem.Prev() {
		keys[i] = elem.Value.key
	}
	return keys
}

func (c *unsafeCache[K, V]) Len() int {
	return c.entries.Len()
}

func (c *unsafeCache[K, V]) Resize(size int) (evicted int) {
	diff := c.Len() - size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		c.removeOldest()
	}
	c.maxEntries = size
	return diff
}

func (c *unsafeCache[K, V]) Clear() {
	for key, elem := range c.bucket {
		if c.onEvicted != nil {
			c.evicting(key, elem.Value.value)
		}
		delete(c.bucket, key)
	}
	c.entries.Init()
}

// removeOldest removes the oldest item from the cache.
func (c *unsafeCache[K, V]) removeOldest() {
	ent := c.entries.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache
func (c *unsafeCache[K, V]) removeElement(elem *list.Element[*entry[K, V]]) {
	c.entries.Remove(elem)
	ent := elem.Value
	delete(c.bucket, ent.key)

	if c.onEvicted == nil {
		return
	}
	c.evicting(ent.key, ent.value)
}

func (c *unsafeCache[K, V]) evicting(key K, value V) {
	if c.async {
		go c.onEvicted(key, value)
	} else {
		c.onEvicted(key, value)
	}
}
