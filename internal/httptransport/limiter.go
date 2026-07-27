package httptransport

import (
	"container/list"
	"sync"
	"time"
)

type peerLimiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	maximum int
	idle    time.Duration
	entries map[string]*list.Element
	recency *list.List
	now     func() time.Time
}

type peerLimitEntry struct {
	key      string
	tokens   float64
	updated  time.Time
	lastSeen time.Time
}

func newPeerLimiter(rate, burst float64, maximum int, idle time.Duration) *peerLimiter {
	return &peerLimiter{
		rate:    rate,
		burst:   burst,
		maximum: maximum,
		idle:    idle,
		entries: make(map[string]*list.Element),
		recency: list.New(),
		now:     time.Now,
	}
}

func (limiter *peerLimiter) allow(key string) bool {
	now := limiter.now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	limiter.removeIdleLocked(now)
	element := limiter.entries[key]
	if element == nil {
		if len(limiter.entries) >= limiter.maximum {
			limiter.evictOldestLocked()
		}
		entry := &peerLimitEntry{
			key:      key,
			tokens:   limiter.burst - 1,
			updated:  now,
			lastSeen: now,
		}
		element = limiter.recency.PushFront(entry)
		limiter.entries[key] = element
		return true
	}

	entry := element.Value.(*peerLimitEntry)
	elapsed := now.Sub(entry.updated).Seconds()
	if elapsed > 0 {
		entry.tokens += elapsed * limiter.rate
		if entry.tokens > limiter.burst {
			entry.tokens = limiter.burst
		}
		entry.updated = now
	}
	entry.lastSeen = now
	limiter.recency.MoveToFront(element)
	if entry.tokens < 1 {
		return false
	}
	entry.tokens--
	return true
}

func (limiter *peerLimiter) removeIdleLocked(now time.Time) {
	for {
		element := limiter.recency.Back()
		if element == nil {
			return
		}
		entry := element.Value.(*peerLimitEntry)
		if now.Sub(entry.lastSeen) < limiter.idle {
			return
		}
		delete(limiter.entries, entry.key)
		limiter.recency.Remove(element)
	}
}

func (limiter *peerLimiter) evictOldestLocked() {
	element := limiter.recency.Back()
	if element == nil {
		return
	}
	entry := element.Value.(*peerLimitEntry)
	delete(limiter.entries, entry.key)
	limiter.recency.Remove(element)
}
