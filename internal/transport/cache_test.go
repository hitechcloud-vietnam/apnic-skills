package transport

import (
	"testing"
	"time"
)

func TestCacheGetSet(t *testing.T) {
	c := newCache(5 * time.Minute)

	// Test miss
	_, ok := c.get("key1")
	if ok {
		t.Error("expected cache miss")
	}

	// Test set and get
	c.set("key1", "value1")
	val, ok := c.get("key1")
	if !ok {
		t.Error("expected cache hit")
	}
	if val.(string) != "value1" {
		t.Errorf("expected value1, got %v", val)
	}
}

func TestCacheExpiry(t *testing.T) {
	c := newCache(1 * time.Nanosecond)
	c.set("key1", "value1")

	// Wait for expiry
	time.Sleep(10 * time.Millisecond)

	_, ok := c.get("key1")
	if ok {
		t.Error("expected cache miss after expiry")
	}
}

func TestCacheOverwrite(t *testing.T) {
	c := newCache(5 * time.Minute)
	c.set("key1", "value1")
	c.set("key1", "value2")

	val, ok := c.get("key1")
	if !ok {
		t.Error("expected cache hit")
	}
	if val.(string) != "value2" {
		t.Errorf("expected value2, got %v", val)
	}
}

func TestCacheMultipleKeys(t *testing.T) {
	c := newCache(5 * time.Minute)
	c.set("key1", "val1")
	c.set("key2", "val2")
	c.set("key3", 42)

	val1, ok1 := c.get("key1")
	val2, ok2 := c.get("key2")
	val3, ok3 := c.get("key3")

	if !ok1 || val1.(string) != "val1" {
		t.Error("key1 failed")
	}
	if !ok2 || val2.(string) != "val2" {
		t.Error("key2 failed")
	}
	if !ok3 || val3.(int) != 42 {
		t.Error("key3 failed")
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := newCache(5 * time.Minute)
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			c.set("key", i)
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			c.get("key")
		}
		done <- true
	}()

	<-done
	<-done
}

func TestCacheKeyConstants(t *testing.T) {
	keys := []string{
		CacheKeyDelegated,
		CacheKeyExtended,
		CacheKeyAssigned,
		CacheKeyLegacy,
		CacheKeyTransfers,
		CacheKeyChanges,
	}
	for _, key := range keys {
		if key == "" {
			t.Error("cache key should not be empty")
		}
	}
}
