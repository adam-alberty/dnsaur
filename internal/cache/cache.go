package cache

import (
	"strings"
	"sync"
	"time"

	"codeberg.org/miekg/dns"
)

type Entry struct {
	Msg       *dns.Msg
	ExpiresAt time.Time
}

type Cache struct {
	mu      sync.RWMutex
	entries map[CacheKey]Entry
}

type CacheKey struct {
	Name  string
	Type  uint16
	Class uint16
}

func New() *Cache {
	return &Cache{
		entries: make(map[CacheKey]Entry),
	}
}

// Adds entry to cache.
func (c *Cache) Set(key CacheKey, msg *dns.Msg, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = Entry{
		Msg:       msg.Copy(),
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Gets response from cache.
func (c *Cache) Get(key CacheKey) (*dns.Msg, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		c.Delete(key)
		return nil, false
	}

	return entry.Msg.Copy(), true
}

// Removes entry from cache.
func (c *Cache) Delete(key CacheKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.entries[key]
	if ok {
		delete(c.entries, key)
	}

	return ok
}

// Creates key from RR.
func Key(q dns.RR) CacheKey {
	header := q.Header()

	return CacheKey{
		Name:  strings.ToLower(header.Name),
		Type:  dns.RRToType(q),
		Class: header.Class,
	}
}
