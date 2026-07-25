// Package concurrency provides bounded deterministic worker coordination for
// internal batch operations.
package concurrency

import (
	"context"
	"runtime"
	"sync"
)

// Options controls bounded ordered processing.
type Options struct {
	// MaxWorkers limits simultaneously executing work. Values <= 0 use the
	// current machine CPU count, capped by the number of items.
	MaxWorkers int
	// ContinueOnCancellation keeps dispatching undispatched items after the
	// parent context is cancelled. Workers still receive the cancelled context.
	// This is useful when callers must populate one result for every input.
	ContinueOnCancellation bool
}

// Stats describes one ordered processing run.
type Stats struct {
	Dispatched   int
	Completed    int
	Committed    int
	StoppedEarly bool
	Cancelled    bool
}

type indexedItem[T any] struct {
	index int
	value T
}

type indexedResult[T any] struct {
	index int
	value T
}

// ProcessOrdered executes work with bounded parallelism and commits completed
// results serially in input order. The commit callback returns false to stop
// dispatching new work. Already running workers are cancelled and drained so
// no goroutines or channel sends are left behind.
func ProcessOrdered[Input, Output any](
	ctx context.Context,
	items []Input,
	options Options,
	work func(context.Context, int, Input) Output,
	commit func(int, Output) bool,
) Stats {
	if work == nil {
		panic("concurrency work function is required")
	}
	if commit == nil {
		panic("concurrency commit function is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if len(items) == 0 {
		return Stats{Cancelled: ctx.Err() != nil}
	}

	workerCount := options.MaxWorkers
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(items) {
		workerCount = len(items)
	}

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	jobs := make(chan indexedItem[Input])
	results := make(chan indexedResult[Output], workerCount)

	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for job := range jobs {
				results <- indexedResult[Output]{
					index: job.index,
					value: work(workerCtx, job.index, job.value),
				}
			}
		}()
	}
	go func() {
		workers.Wait()
		close(results)
	}()

	stats := Stats{}
	nextDispatch := 0
	nextCommit := 0
	jobsClosed := false
	dispatching := true
	committing := true

	closeJobs := func() {
		if jobsClosed {
			return
		}
		close(jobs)
		jobsClosed = true
		dispatching = false
	}
	dispatch := func() {
		jobs <- indexedItem[Input]{index: nextDispatch, value: items[nextDispatch]}
		nextDispatch++
		stats.Dispatched++
		if nextDispatch == len(items) {
			closeJobs()
		}
	}

	for range workerCount {
		dispatch()
		if jobsClosed {
			break
		}
	}

	pending := make(map[int]Output, workerCount)
	for result := range results {
		stats.Completed++
		pending[result.index] = result.value

		if !options.ContinueOnCancellation && ctx.Err() != nil && dispatching {
			stats.Cancelled = true
			closeJobs()
		}

		for committing {
			value, ok := pending[nextCommit]
			if !ok {
				break
			}
			delete(pending, nextCommit)

			continueProcessing := commit(nextCommit, value)
			stats.Committed++
			nextCommit++
			if !continueProcessing {
				stats.StoppedEarly = true
				committing = false
				if dispatching {
					closeJobs()
				}
				cancelWorkers()
				break
			}

			if dispatching && !options.ContinueOnCancellation && ctx.Err() != nil {
				stats.Cancelled = true
				closeJobs()
			}
			if dispatching && nextDispatch < len(items) {
				dispatch()
			}
		}
	}

	if ctx.Err() != nil {
		stats.Cancelled = true
	}
	return stats
}
