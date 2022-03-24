package asyncjob

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// AsyncJob is a representation of AsyncJob processus
type AsyncJob[T any] struct {
	mu             sync.Mutex
	wg             sync.WaitGroup
	workers        int
	onJob          func(job Job[T]) error
	onProgressFunc func(progress Progress)
	jobs           []T
	position       int
	queueJob       chan Job[T]
	errors         chan error
	startTimer     time.Time
}

// New allows you to retrieve a new instance of AsyncJob
func New[T any]() *AsyncJob[T] {
	// new instance of AsyncJob

	aj := new(AsyncJob[T])
	// set default concurrency with num cpu value
	aj.SetWorkers(runtime.NumCPU())
	// create channel for jobs error
	aj.errors = make(chan error)

	return aj
}

// SetWorkers allows you to set the number of asynchronous jobs
func (aj *AsyncJob[T]) SetWorkers(workers int) *AsyncJob[T] {
	aj.workers = workers
	return aj
}

// GetWorkers allows you to retrieve the number of workers
func (aj *AsyncJob[T]) GetWorkers() int {
	return aj.workers
}

// Run allows you to start the process
func (aj *AsyncJob[T]) Run(listener func(job Job[T]) error, data []T) (err error) {
	aj.jobs = data
	aj.onJob = listener

	return aj._Process()
}

// OnProgress allows you set callback function for ETA
func (aj *AsyncJob[T]) OnProgress(onProgressFunc func(progress Progress)) *AsyncJob[T] {
	aj.onProgressFunc = onProgressFunc

	// return current instance for chaining method
	return aj
}
func (aj *AsyncJob[T]) _Progress() {
	// we save time if the anonymous function is not initialized
	if aj.onProgressFunc == nil {
		return
	}
	// we secure the access struct data
	aj.mu.Lock()
	defer aj.mu.Unlock()

	// division by zero security
	if time.Since(aj.startTimer).Milliseconds() == 0 {
		return
	}
	// jobs finished
	size := len(aj.jobs)
	if size == aj.position {
		return
	}
	// we calculate the remaining jobs
	ret := size - aj.position

	// call the anonymous function with data
	aj.onProgressFunc(Progress{aj.position, size, time.Duration((time.Since(aj.startTimer).Nanoseconds()/int64(aj.position))*int64(ret)) * time.Nanosecond})
}

// _Next allows you to retrieve the next job
func (aj *AsyncJob[T]) _Next() {
	aj.mu.Lock()
	defer aj.mu.Unlock()

	if aj.position == len(aj.jobs) {
		return
	}
	aj.wg.Add(1)

	aj.queueJob <- Job[T]{index: aj.position, data: aj.jobs[aj.position]}
	aj.position = aj.position + 1
}
func (aj *AsyncJob[T]) _SetError(err error) {
	aj.errors <- err
}
func (aj *AsyncJob[T]) _Process() error {
	// if jobs is empty.
	if len(aj.jobs) == 0 {
		return nil
	}
	// start timer
	aj.startTimer = time.Now()
	// create channel for jobs
	aj.queueJob = make(chan Job[T])

	// create channel for wait group
	waitCh := make(chan bool, 1)
	go func() {
		// for each job into the queue
		for job := range aj.queueJob {
			go func(job Job[T]) {
				defer aj.wg.Done()
				// call the anonymous function for recovered error
				defer func() {
					if v := recover(); v != nil {
						recoverErr, ok := v.(error)
						if !ok {
							recoverErr = fmt.Errorf("%s", v)
						}
						// we set the error
						aj._SetError(recoverErr)
						return
					}
				}()
				// call the anonymous function for job
				err := aj.onJob(job)
				if err != nil {
					// we set the error
					aj._SetError(err)
					return
				}
				// we notify the jobs progress
				aj._Progress()
				// we call the next job
				aj._Next()
			}(job)
		}
	}()

	// the number of jobs for the start-up is calculated
	max := aj.GetWorkers()
	size := len(aj.jobs)
	if size <= max {
		max = size
	}
	// we start the workers
	for i := 0; i < max; i++ {
		aj._Next()
	}

	// we wait for the end of the jobs
	go func() {
		aj.wg.Wait()
		close(waitCh)
	}()

	// we check the errors
	var err error
	select {
	case v := <-aj.errors:
		err = v
	case <-waitCh:
		break
	}

	// return the error
	return err
}
