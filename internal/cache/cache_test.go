package cache_test

import (
	"testing"
	"time"

	"github.com/sperax/flight-price-service/internal/cache"
)

func TestCache_SetAndGet(t *testing.T) {
	c := cache.New[string, string](5 * time.Second)
	c.Set("key", "value")

	got, ok := c.Get("key")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if got != "value" {
		t.Errorf("expected value, got %s", got)
	}
}

func TestCache_Miss(t *testing.T) {
	c := cache.New[string, int](5 * time.Second)

	_, ok := c.Get("missing")
	if ok {
		t.Error("expected cache miss, got hit")
	}
}

func TestCache_Expiry(t *testing.T) {
	c := cache.New[string, string](50 * time.Millisecond)
	c.Set("key", "value")

	time.Sleep(100 * time.Millisecond)

	_, ok := c.Get("key")
	if ok {
		t.Error("expected cache miss after TTL, got hit")
	}
}

func TestCache_OverwriteResetsExpiry(t *testing.T) {
	c := cache.New[string, string](200 * time.Millisecond)
	c.Set("key", "first")

	time.Sleep(100 * time.Millisecond)
	c.Set("key", "second") // reset TTL

	time.Sleep(150 * time.Millisecond) // total 250ms but TTL reset at 100ms, so still valid

	got, ok := c.Get("key")
	if !ok {
		t.Fatal("expected cache hit after TTL reset, got miss")
	}
	if got != "second" {
		t.Errorf("expected second, got %s", got)
	}
}

func TestCache_DifferentKeys(t *testing.T) {
	c := cache.New[string, int](5 * time.Second)
	c.Set("a", 1)
	c.Set("b", 2)

	v1, ok1 := c.Get("a")
	v2, ok2 := c.Get("b")

	if !ok1 || v1 != 1 {
		t.Errorf("expected a=1, got ok=%v val=%d", ok1, v1)
	}
	if !ok2 || v2 != 2 {
		t.Errorf("expected b=2, got ok=%v val=%d", ok2, v2)
	}
}
