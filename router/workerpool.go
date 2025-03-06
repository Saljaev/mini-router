package router

import (
	"net/http"
	"sync"
)

type Job struct {
	Context     *APIContext
	HandlerFunc func(ctx *APIContext)
}

type WorkerPool struct {
	jobs chan Job // pool of jobs

	wg sync.WaitGroup

	workers int // count of max running in time goroutines
}

// NewWorkerPool create WorkerPool with count of workers and
// size of channel jobs
func NewWorkerPool(workers, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		jobs:    make(chan Job, queueSize),
		workers: workers,
	}

	for i := 0; i < workers; i++ {
		wp.startWorker()
	}

	return wp
}

func (wp *WorkerPool) startWorker() {
	wp.wg.Add(1)

	go func() {
		defer wp.wg.Done()

		for job := range wp.jobs {
			if job.Context != nil && job.Context.Err() == nil {
				job.HandlerFunc(job.Context)
			}
		}
	}()
}

// Submit add new job in pool
// Wrap all handlers in one func
func (wp *WorkerPool) Submit(ctx *APIContext, handlers []HandlerFunc) {
	ctx.wg.Add(1)

	wrappedHandler := func(ctx *APIContext) {
		defer ctx.wg.Done()

		for _, h := range handlers {
			select {
			case <-ctx.Done():
				ctx.Error("context canceled", ErrContextTimeout)
				ctx.WriteFailure(http.StatusGatewayTimeout, "request timeout")
				return
			default:
				h(ctx)
			}
		}
	}

	wp.jobs <- Job{Context: ctx, HandlerFunc: wrappedHandler}
}

// Shutdown close jobs channel and wait for all goroutines done
func (wp *WorkerPool) Shutdown() {
	close(wp.jobs)
	wp.wg.Wait()
}
