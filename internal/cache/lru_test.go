package cache

import (
	"sync"
	"testing"
)

func TestLRUBasicOperations(t *testing.T) {
	c := New(3)

	// Test Set and Get
	c.Set(1, "one")
	c.Set(2, "two")
	c.Set(3, "three")

	if v, ok := c.Get(1); !ok || v != "one" {
		t.Errorf("Get(1) = %q, %v; want 'one', true", v, ok)
	}

	if c.Len() != 3 {
		t.Errorf("Len() = %d; want 3", c.Len())
	}
}

func TestLRUEviction(t *testing.T) {
	c := New(2)

	c.Set(1, "one")
	c.Set(2, "two")
	c.Set(3, "three") // Should evict key 1

	if _, ok := c.Get(1); ok {
		t.Error("key 1 should have been evicted")
	}

	if v, ok := c.Get(2); !ok || v != "two" {
		t.Errorf("Get(2) = %q, %v; want 'two', true", v, ok)
	}

	if v, ok := c.Get(3); !ok || v != "three" {
		t.Errorf("Get(3) = %q, %v; want 'three', true", v, ok)
	}
}

func TestLRUAccessOrder(t *testing.T) {
	c := New(2)

	c.Set(1, "one")
	c.Set(2, "two")

	// Access key 1 to make it recently used
	c.Get(1)

	// Add key 3, should evict key 2 (least recently used)
	c.Set(3, "three")

	if _, ok := c.Get(2); ok {
		t.Error("key 2 should have been evicted")
	}

	if _, ok := c.Get(1); !ok {
		t.Error("key 1 should still exist")
	}

	if _, ok := c.Get(3); !ok {
		t.Error("key 3 should exist")
	}
}

func TestLRUUpdate(t *testing.T) {
	c := New(2)

	c.Set(1, "one")
	c.Set(1, "ONE") // Update

	if v, ok := c.Get(1); !ok || v != "ONE" {
		t.Errorf("Get(1) = %q, %v; want 'ONE', true", v, ok)
	}

	if c.Len() != 1 {
		t.Errorf("Len() = %d; want 1", c.Len())
	}
}

func TestLRUClear(t *testing.T) {
	c := New(3)

	c.Set(1, "one")
	c.Set(2, "two")
	c.Clear()

	if c.Len() != 0 {
		t.Errorf("Len() after Clear() = %d; want 0", c.Len())
	}

	if _, ok := c.Get(1); ok {
		t.Error("key 1 should not exist after Clear()")
	}
}

func TestLRUConcurrency(t *testing.T) {
	c := New(100)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Set(base*100+j, "value")
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Get(base*100 + j)
			}
		}(i)
	}

	wg.Wait()

	// Verify no panic and reasonable state
	if c.Len() > 100 {
		t.Errorf("Len() = %d; should not exceed capacity 100", c.Len())
	}
}

func TestLRUMiss(t *testing.T) {
	c := New(2)

	if v, ok := c.Get(999); ok {
		t.Errorf("Get(999) = %q, %v; want '', false", v, ok)
	}
}

func TestLRUZeroCapacity(t *testing.T) {
	// Should default to capacity 1
	c := New(0)

	c.Set(1, "one")
	c.Set(2, "two") // Should evict key 1

	if _, ok := c.Get(1); ok {
		t.Error("key 1 should have been evicted with capacity 1")
	}

	if v, ok := c.Get(2); !ok || v != "two" {
		t.Errorf("Get(2) = %q, %v; want 'two', true", v, ok)
	}
}

func TestLRUUpdateMovesToFront(t *testing.T) {
	c := New(2)

	c.Set(1, "one")
	c.Set(2, "two")

	// Update key 1, making it most recently used
	c.Set(1, "ONE")

	// Add key 3, should evict key 2
	c.Set(3, "three")

	if _, ok := c.Get(2); ok {
		t.Error("key 2 should have been evicted")
	}

	if v, ok := c.Get(1); !ok || v != "ONE" {
		t.Errorf("Get(1) = %q, %v; want 'ONE', true", v, ok)
	}
}

func TestLRUEvictionOrder(t *testing.T) {
	c := New(3)

	// Fill cache
	c.Set(1, "one")
	c.Set(2, "two")
	c.Set(3, "three")

	// Access in order: 2, 1, 3 (making 3 most recent, 2 least recent)
	c.Get(2)
	c.Get(1)
	c.Get(3)

	// Add new item, should evict 2
	c.Set(4, "four")

	if _, ok := c.Get(2); ok {
		t.Error("key 2 should have been evicted (was least recently used)")
	}

	// Verify others still exist
	for _, key := range []int{1, 3, 4} {
		if _, ok := c.Get(key); !ok {
			t.Errorf("key %d should still exist", key)
		}
	}
}
