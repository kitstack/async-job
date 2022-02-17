package asyncjob

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"time"
)

// AsyncJob is a representation of AsyncJob processus
type AsyncJob struct {
	sync.Mutex
	sync.WaitGroup
	workers        int
	onJob          func(job Job) error
	onProgressFunc func(progress Progress)
	jobs           reflect.Value
	position       int
	queueJob       chan Job
	errors         chan error
	startTimer     time.Time
}

// New allows you to retrieve a new instance of AsyncJob
func New() *AsyncJob {
	// new instance of AsyncJob
	aj := new(AsyncJob)
	// set default concurrency with num cpu value
	aj.SetWorkers(runtime.NumCPU())
	// create channel for jobs error
	aj.errors = make(chan error)

	return aj
}

// SetWorkers allows you to set the number of asynchronous jobs
func (aj *AsyncJob) SetWorkers(concurrency int) *AsyncJob {
	aj.workers = concurrency
	return aj
}

// GetWorkers allows you to retrieve the number of workers
func (aj *AsyncJob) GetWorkers() int {
	return aj.workers
}

// Run allows you to start the process
func (aj *AsyncJob) Run(listener func(job Job) error, data interface{}) (err error) {
	s := reflect.ValueOf(data)
	if s.Kind() != reflect.Slice {
		return fmt.Errorf("invalid input data : got %v want: slice", s.Kind())
	}
	if listener == nil {
		return fmt.Errorf("the listener cannot be null because it is used to process jobs")
	}
	aj.jobs = s
	aj.onJob = listener

	return aj._Process()
}

// OnProgress allows you set callback function for ETA
func (aj *AsyncJob) OnProgress(onProgressFunc func(progress Progress)) *AsyncJob {
	aj.onProgressFunc = onProgressFunc

	// return current instance for chaining method
	return aj
}
func (aj *AsyncJob) _Progress() {
	// we save time if the anonymous function is not initialized
	if aj.onProgressFunc == nil {
		return
	}
	// we secure the access struct data
	aj.Lock()
	defer aj.Unlock()

	// division by zero security
	if time.Since(aj.startTimer).Milliseconds() == 0 {
		return
	}
	// jobs finished
	if aj.jobs.Len() == aj.position {
		return
	}
	// we calculate the remaining jobs
	ret := aj.jobs.Len() - aj.position

	// call the anonymous function with data
	aj.onProgressFunc(Progress{aj.position, aj.jobs.Len(), time.Duration((time.Since(aj.startTimer).Milliseconds()/int64(aj.position))*int64(ret)) * time.Millisecond})
}

// _Next allows you to retrieve the next job
func (aj *AsyncJob) _Next() {
	aj.Lock()
	defer aj.Unlock()

	if aj.position == aj.jobs.Len() {
		return
	}
	aj.Add(1)
	aj.queueJob <- Job{index: aj.position, data: aj.jobs.Index(aj.position).Interface()}
	aj.position = aj.position + 1
}
func (aj *AsyncJob) _SetError(err error) {
	aj.errors <- err
}
func (aj *AsyncJob) _Process() error {
	// if jobs is empty.
	if aj.jobs.Len() == 0 {
		return nil
	}

	// start timer
	aj.startTimer = time.Now()
	// create channel for jobs
	aj.queueJob = make(chan Job)

	// create channel for wait group
	waitCh := make(chan bool, 1)
	go func() {
		// for each job into the queue
		for job := range aj.queueJob {
			go func(job Job) {
				defer aj.Done()
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
	size := aj.jobs.Len()
	if size <= max {
		max = size
	}
	// we start the workers
	for i := 0; i < max; i++ {
		aj._Next()
	}

	// we wait for the end of the jobs
	go func() {
		aj.Wait()
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
