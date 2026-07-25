package concurrency

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestProcessOrderedCommitsInInputOrder(t *testing.T) {
	items := []int{0, 1, 2, 3}
	gates := make([]chan struct{}, len(items))
	for index := range gates {
		gates[index] = make(chan struct{})
	}
	started := make(chan int, len(items))
	committed := make([]int, 0, len(items))
	done := make(chan Stats, 1)

	go func() {
		done <- ProcessOrdered(context.Background(), items, Options{MaxWorkers: len(items)}, func(_ context.Context, index, item int) int {
			started <- index
			<-gates[index]
			return item
		}, func(_ int, result int) bool {
			committed = append(committed, result)
			return true
		})
	}()

	waitForSignals(t, started, len(items))
	for index := len(gates) - 1; index >= 0; index-- {
		close(gates[index])
	}

	stats := waitForStats(t, done)
	if !reflect.DeepEqual(committed, items) {
		t.Fatalf("committed = %v, want %v", committed, items)
	}
	if stats.Dispatched != len(items) || stats.Completed != len(items) || stats.Committed != len(items) {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestProcessOrderedBoundsInFlightWork(t *testing.T) {
	const workers = 3
	items := make([]int, 64)
	started := make(chan int, len(items))
	release := make(chan struct{})
	var active atomic.Int32
	var maximum atomic.Int32
	done := make(chan Stats, 1)

	go func() {
		done <- ProcessOrdered(context.Background(), items, Options{MaxWorkers: workers}, func(_ context.Context, index, item int) int {
			current := active.Add(1)
			for {
				observed := maximum.Load()
				if current <= observed || maximum.CompareAndSwap(observed, current) {
					break
				}
			}
			started <- index
			<-release
			active.Add(-1)
			return item
		}, func(_ int, _ int) bool {
			return true
		})
	}()

	waitForSignals(t, started, workers)
	if got := active.Load(); got != workers {
		t.Fatalf("active workers = %d, want %d", got, workers)
	}
	close(release)

	stats := waitForStats(t, done)
	if got := maximum.Load(); got > workers {
		t.Fatalf("maximum in-flight work = %d, want <= %d", got, workers)
	}
	if stats.Committed != len(items) {
		t.Fatalf("committed = %d, want %d", stats.Committed, len(items))
	}
}

func TestProcessOrderedStopsDispatchAfterCancellation(t *testing.T) {
	const workers = 4
	items := make([]int, 32)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan int, workers)
	done := make(chan Stats, 1)

	go func() {
		done <- ProcessOrdered(ctx, items, Options{MaxWorkers: workers}, func(ctx context.Context, index, item int) int {
			started <- index
			<-ctx.Done()
			return item
		}, func(_ int, _ int) bool {
			return true
		})
	}()

	waitForSignals(t, started, workers)
	cancel()
	stats := waitForStats(t, done)

	if !stats.Cancelled {
		t.Fatalf("cancelled = false, want true: %+v", stats)
	}
	if stats.Dispatched != workers {
		t.Fatalf("dispatched = %d, want %d", stats.Dispatched, workers)
	}
	if stats.Committed != workers {
		t.Fatalf("committed = %d, want %d", stats.Committed, workers)
	}
}

func TestProcessOrderedCanCompleteEveryPositionAfterCancellation(t *testing.T) {
	items := make([]int, 20)
	for index := range items {
		items[index] = index
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	committed := make([]int, 0, len(items))
	var observedLiveContext atomic.Bool

	stats := ProcessOrdered(ctx, items, Options{
		MaxWorkers:             3,
		ContinueOnCancellation: true,
	}, func(ctx context.Context, _ int, item int) int {
		if ctx.Err() == nil {
			observedLiveContext.Store(true)
		}
		return item
	}, func(_ int, result int) bool {
		committed = append(committed, result)
		return true
	})

	if observedLiveContext.Load() {
		t.Fatal("worker context was not cancelled")
	}
	if !reflect.DeepEqual(committed, items) {
		t.Fatalf("committed = %v, want %v", committed, items)
	}
	if stats.Dispatched != len(items) || stats.Committed != len(items) {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestProcessOrderedStopsAfterCommitDecision(t *testing.T) {
	items := make([]int, 20)
	stats := ProcessOrdered(context.Background(), items, Options{MaxWorkers: 4}, func(_ context.Context, index, _ int) int {
		return index
	}, func(index int, _ int) bool {
		return index < 2
	})

	if !stats.StoppedEarly {
		t.Fatalf("stoppedEarly = false, want true: %+v", stats)
	}
	if stats.Committed != 3 {
		t.Fatalf("committed = %d, want 3", stats.Committed)
	}
	if stats.Dispatched > 6 {
		t.Fatalf("dispatched = %d, want bounded window <= 6", stats.Dispatched)
	}
}

func waitForSignals(t *testing.T, signals <-chan int, count int) {
	t.Helper()
	for range count {
		select {
		case <-signals:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %d worker signals", count)
		}
	}
}

func waitForStats(t *testing.T, done <-chan Stats) Stats {
	t.Helper()
	select {
	case stats := <-done:
		return stats
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ordered processing")
		return Stats{}
	}
}
