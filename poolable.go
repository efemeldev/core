package main

import "sync"

type PoolableFunc[T any] func(T)

type PoolableFuncWrapper[T any] func(maxWorkers int, workChannel <-chan T)

type PoolableInit[T any] struct {
	Run  PoolableFuncWrapper[T]
	Wait func()
}

func Poolable[T any](workerFunc PoolableFunc[T]) *PoolableInit[T] {

	// create wg
	var wg sync.WaitGroup

	run := func(maxWorkers int, workChannel <-chan T) {
		// Create a channel to control the number of concurrent workers
		workerChannel := make(chan struct{}, maxWorkers)

		for work := range workChannel {
			// Acquire a worker slot
			workerChannel <- struct{}{}

			// Increment the wait group counter
			wg.Add(1)

			// Launch a goroutine to process the work
			go func(work T) {
				defer func() {
					// Release the worker slot
					<-workerChannel

					// Decrement the wait group counter when the goroutine exits
					wg.Done()
				}()

				// Execute the worker function
				workerFunc(work)
			}(work)
		}
	}

	return &PoolableInit[T]{
		Run: run,
		Wait: func() {
			wg.Wait()
		},
	}
}
