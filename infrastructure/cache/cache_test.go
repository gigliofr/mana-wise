package cache_test

import (
	"testing"
	"time"

	"github.com/gigliofr/mana-wise/infrastructure/cache"
)

func TestCache_SetAndGet(t *testing.T) {
	c := cache.New()
	c.Set("key1", "value1", 1*time.Minute)

	v, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to be present")
	}
	if v.(string) != "value1" {
		t.Errorf("expected 'value1', got %v", v)
	}
}

func TestCache_MissOnUnknownKey(t *testing.T) {
	c := cache.New()
	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected miss for unknown key")
	}
}

func TestCache_ExpiryEviction(t *testing.T) {
	c := cache.New()
	c.Set("short", "v", 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	_, ok := c.Get("short")
	if ok {
		t.Error("expected expired key to be evicted")
	}
}

func TestCache_Delete(t *testing.T) {
	c := cache.New()
	c.Set("del", 42, 1*time.Minute)
	c.Delete("del")

	_, ok := c.Get("del")
	if ok {
		t.Error("expected deleted key to be absent")
	}
}

func TestCache_OverwriteValue(t *testing.T) {
	c := cache.New()
	c.Set("k", "first", 1*time.Minute)
	c.Set("k", "second", 1*time.Minute)

	v, ok := c.Get("k")
	if !ok {
		t.Fatal("key should exist")
	}
	if v.(string) != "second" {
		t.Errorf("expected 'second', got %v", v)
	}
}

func TestCache_Janitor(t *testing.T) {
	c := cache.New()
	done := make(chan struct{})
	defer close(done)

	c.StartJanitor(10*time.Millisecond, done)
	c.Set("jk", "jv", 5*time.Millisecond)
	time.Sleep(30 * time.Millisecond)

	_, ok := c.Get("jk")
	if ok {
		t.Error("janitor should have evicted the expired entry")
	}
}
