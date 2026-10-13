package search

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello world"},
		{"  FOO   BAR  ", "foo bar"},
		{"\t\nMultiple \t  Spaces\n\n", "multiple spaces"},
		{"", ""},
		{"   ", ""},
		{"already clean", "already clean"},
		{"GENSHIN IMPACT columbina", "genshin impact columbina"},
	}

	for _, tt := range tests {
		got := NormalizeQuery(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNewCache_Defaults(t *testing.T) {
	// Zero TTL should default to 20 minutes, default max entries
	c1 := NewCache(0)
	defer c1.Close()
	if c1.ttl != 20*time.Minute {
		t.Errorf("expected default ttl 20m, got %v", c1.ttl)
	}
	if c1.maxEntries != DefaultMaxEntries {
		t.Errorf("expected default max entries %d, got %d", DefaultMaxEntries, c1.maxEntries)
	}

	// Custom TTL and custom maxEntries
	c2 := NewCache(5*time.Minute, 50)
	defer c2.Close()
	if c2.ttl != 5*time.Minute {
		t.Errorf("expected ttl 5m, got %v", c2.ttl)
	}
	if c2.maxEntries != 50 {
		t.Errorf("expected max entries 50, got %d", c2.maxEntries)
	}
}

func TestCache_GetSet(t *testing.T) {
	c := NewCache(1 * time.Minute)
	defer c.Close()

	res := SearchResult{
		Query:  "genshin columbina",
		Answer: "Columbina is the Damselette.",
		Results: []SearchResultItem{
			{Title: "Columbina Lore", URL: "https://example.com", Content: "Lore text"},
		},
	}

	// Set with mixed case and extra spaces
	c.Set("  Genshin   Columbina  ", res)

	if c.Len() != 1 {
		t.Fatalf("expected Len() == 1, got %d", c.Len())
	}

	// Retrieve with different case and spacing
	got, found := c.Get("genshin columbina")
	if !found {
		t.Fatal("expected item to be found in cache")
	}
	if got.Answer != res.Answer {
		t.Errorf("got Answer = %q, want %q", got.Answer, res.Answer)
	}
	if len(got.Results) != 1 || got.Results[0].Title != "Columbina Lore" {
		t.Errorf("got Results mismatch: %+v", got.Results)
	}

	// Empty query Get and Set
	c.Set("", SearchResult{})
	if c.Len() != 1 {
		t.Errorf("setting empty query should not increase cache size")
	}
	_, foundEmpty := c.Get("")
	if foundEmpty {
		t.Errorf("empty query should return false from Get")
	}
}

func TestCache_ExpiryAndPurge(t *testing.T) {
	shortTTL := 50 * time.Millisecond
	c := NewCache(shortTTL)
	defer c.Close()

	c.Set("expire-me", SearchResult{Answer: "temporary"})

	got, found := c.Get("expire-me")
	if !found || got.Answer != "temporary" {
		t.Fatal("expected item to be immediately available")
	}

	time.Sleep(70 * time.Millisecond)

	// Get should return false once expired
	_, foundAfter := c.Get("expire-me")
	if foundAfter {
		t.Error("expected item to be expired and not found via Get")
	}

	// Item is still in memory until purged
	if c.Len() != 1 {
		t.Errorf("expected Len() to be 1 prior to purge, got %d", c.Len())
	}

	purged := c.PurgeExpired()
	if purged != 1 {
		t.Errorf("expected PurgeExpired() == 1, got %d", purged)
	}
	if c.Len() != 0 {
		t.Errorf("expected Len() == 0 after purge, got %d", c.Len())
	}
}

func TestCache_Delete(t *testing.T) {
	c := NewCache(1 * time.Minute)
	defer c.Close()

	c.Set("key1", SearchResult{Answer: "val1"})
	c.Set("key2", SearchResult{Answer: "val2"})
	if c.Len() != 2 {
		t.Fatalf("expected 2 items, got %d", c.Len())
	}

	c.Delete("  KEY1 ")
	if c.Len() != 1 {
		t.Errorf("expected 1 item after delete, got %d", c.Len())
	}

	_, found := c.Get("key1")
	if found {
		t.Error("key1 should have been deleted")
	}

	// Deleting empty query or non-existent key is a no-op
	c.Delete("")
	c.Delete("non-existent")
	if c.Len() != 1 {
		t.Errorf("expected 1 item remaining, got %d", c.Len())
	}
}

func TestCache_CapacityEviction(t *testing.T) {
	// Max capacity = 2
	c := NewCache(1*time.Minute, 2)
	defer c.Close()

	c.Set("item1", SearchResult{Answer: "1"})
	time.Sleep(10 * time.Millisecond)
	c.Set("item2", SearchResult{Answer: "2"})

	if c.Len() != 2 {
		t.Fatalf("expected 2 items, got %d", c.Len())
	}

	// Add 3rd item -> oldest (item1) should be evicted
	time.Sleep(10 * time.Millisecond)
	c.Set("item3", SearchResult{Answer: "3"})

	if c.Len() != 2 {
		t.Fatalf("expected cache size to stay at max 2, got %d", c.Len())
	}

	if _, found := c.Get("item1"); found {
		t.Errorf("item1 should have been evicted")
	}
	if _, found := c.Get("item2"); !found {
		t.Errorf("item2 should still be cached")
	}
	if _, found := c.Get("item3"); !found {
		t.Errorf("item3 should be cached")
	}
}

func TestCache_Concurrency(t *testing.T) {
	c := NewCache(100*time.Millisecond, 20)
	defer c.Close()

	var wg sync.WaitGroup
	workers := 10
	iterations := 50

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := fmt.Sprintf("query-%d-%d", id, j%5)
				c.Set(key, SearchResult{Answer: fmt.Sprintf("ans-%d", j)})
				c.Get(key)
				if j%10 == 0 {
					c.PurgeExpired()
					c.Delete(key)
				}
				_ = c.Len()
			}
		}(i)
	}

	wg.Wait()
}

func TestCache_CloseIdempotency(t *testing.T) {
	c := NewCache(1 * time.Minute)
	// Calling Close multiple times should not panic or error
	c.Close()
	c.Close()
}
