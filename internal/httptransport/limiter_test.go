package httptransport

import (
	"testing"
	"time"
)

func TestPeerLimiterEnforcesBurstAndRefill(t *testing.T) {
	now := time.Unix(100, 0)
	limiter := newPeerLimiter(2, 2, 4, time.Minute)
	limiter.now = func() time.Time { return now }

	if !limiter.allow("peer") {
		t.Fatal("first request in the initial burst was rejected")
	}
	if !limiter.allow("peer") {
		t.Fatal("second request in the initial burst was rejected")
	}
	if limiter.allow("peer") {
		t.Fatal("request beyond burst was accepted")
	}
	now = now.Add(500 * time.Millisecond)
	if !limiter.allow("peer") {
		t.Fatal("token did not refill")
	}
}

func TestPeerLimiterBoundsAndExpiresState(t *testing.T) {
	now := time.Unix(100, 0)
	limiter := newPeerLimiter(1, 1, 2, time.Minute)
	limiter.now = func() time.Time { return now }

	if !limiter.allow("one") || !limiter.allow("two") || !limiter.allow("three") {
		t.Fatal("new peer was rejected")
	}
	if len(limiter.entries) != 2 {
		t.Fatalf("entry count = %d, want 2", len(limiter.entries))
	}
	if _, ok := limiter.entries["one"]; ok {
		t.Fatal("least-recently-used peer was not evicted")
	}

	now = now.Add(2 * time.Minute)
	if !limiter.allow("four") {
		t.Fatal("request after idle cleanup was rejected")
	}
	if len(limiter.entries) != 1 {
		t.Fatalf("idle cleanup left %d entries", len(limiter.entries))
	}
}
