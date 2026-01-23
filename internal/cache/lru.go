package cache

import (
	"sync"
)

// node is a doubly-linked list node for LRU tracking
type node struct {
	key   int
	value string
	prev  *node
	next  *node
}

// LRU is a thread-safe least-recently-used cache for shared strings
type LRU struct {
	mu       sync.RWMutex
	capacity int
	items    map[int]*node
	head     *node // Most recently used
	tail     *node // Least recently used
}

// New creates a new LRU cache with the given capacity.
// If capacity is less than 1, it defaults to 1.
func New(capacity int) *LRU {
	if capacity < 1 {
		capacity = 1
	}
	return &LRU{
		capacity: capacity,
		items:    make(map[int]*node),
	}
}

// Get retrieves a value by key, returning (value, true) if found.
// This operation marks the item as recently used.
func (c *LRU) Get(key int) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	n, ok := c.items[key]
	if !ok {
		return "", false
	}

	// Move to front (most recently used)
	c.moveToFront(n)
	return n.value, true
}

// Set adds or updates a key-value pair in the cache.
// If the key exists, it updates the value and marks it as recently used.
// If adding a new key exceeds capacity, the least recently used item is evicted.
func (c *LRU) Set(key int, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing node
	if n, ok := c.items[key]; ok {
		n.value = value
		c.moveToFront(n)
		return
	}

	// Create new node
	n := &node{key: key, value: value}
	c.items[key] = n
	c.addToFront(n)

	// Evict if over capacity
	if len(c.items) > c.capacity {
		c.evict()
	}
}

// Len returns the current number of items in the cache.
func (c *LRU) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all items from the cache.
func (c *LRU) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[int]*node)
	c.head = nil
	c.tail = nil
}

// addToFront adds a node to the front of the doubly-linked list.
func (c *LRU) addToFront(n *node) {
	n.prev = nil
	n.next = c.head

	if c.head != nil {
		c.head.prev = n
	}
	c.head = n

	if c.tail == nil {
		c.tail = n
	}
}

// removeNode removes a node from the doubly-linked list.
func (c *LRU) removeNode(n *node) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		c.head = n.next
	}

	if n.next != nil {
		n.next.prev = n.prev
	} else {
		c.tail = n.prev
	}
}

// moveToFront moves an existing node to the front of the list.
func (c *LRU) moveToFront(n *node) {
	if n == c.head {
		return
	}
	c.removeNode(n)
	c.addToFront(n)
}

// evict removes the least recently used item (tail).
func (c *LRU) evict() {
	if c.tail == nil {
		return
	}
	delete(c.items, c.tail.key)
	c.removeNode(c.tail)
}
