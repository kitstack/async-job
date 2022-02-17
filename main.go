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
	concurrency         int
	onJob               func(job Job) error
	onEtaFunc           func(current int, total int, estimateTimeLeft time.Duration)
	jobs                reflect.Value
	position            int
	queueJob            chan Job
	waitingCompleteJobs chan bool
	queueUsage          int
	errors              chan error
	eta                 time.Time
}

// New allows you to retrieve a new instance of AsyncJob
func New() *AsyncJob {
	// new instance of AsyncJob
	aj := new(AsyncJob)
	// set default concurrency with num cpu value
	aj.SetConcurrency(runtime.NumCPU())
	// create channel for jobs error
	aj.errors = make(chan error)

	return aj
}

// SetConcurrency allows you to set the number of asynchronous jobs
func (aj *AsyncJob) SetConcurrency(concurrency int) *AsyncJob {
	aj.concurrency = concurrency
	return aj
}

// GetConcurrency allows you to retrieve the number and value of asynchronous tasks
func (aj *AsyncJob) GetConcurrency() int {
	return aj.concurrency
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

// onEta allows you set callback function for ETA
func (aj *AsyncJob) onEta(eta func(current int, total int, estimateTimeLeft time.Duration)) {
	aj.onEtaFunc = eta
}
func (aj *AsyncJob) _Eta() {
	// we save time if the anonymous function is not initialized
	if aj.onEtaFunc == nil {
		return
	}
	// we secure the access struct data
	aj.Lock()
	defer aj.Unlock()

	// division by zero security
	if time.Since(aj.eta).Milliseconds() == 0 {
		return
	}
	// jobs finished
	if aj.jobs.Len() == aj.position {
		return
	}
	// we calculate the remaining jobs
	ret := aj.jobs.Len() - aj.position

	// call the anonymous function with data
	aj.onEtaFunc(aj.position, aj.jobs.Len(), time.Duration((time.Since(aj.eta).Milliseconds()/int64(aj.position))*int64(ret))*time.Millisecond)
}
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

	aj.eta = time.Now()
	aj.queueJob = make(chan Job)
	aj.waitingCompleteJobs = make(chan bool, 1)

	waitCh := make(chan bool, 1)
	go func() {
		for job := range aj.queueJob {
			go func(job Job) {
				defer aj.Done()
				defer func() {
					if v := recover(); v != nil {
						recoverErr, ok := v.(error)
						if !ok {
							recoverErr = fmt.Errorf("%s", v)
						}
						aj._SetError(recoverErr)
						return
					}
				}()
				err := aj.onJob(job)
				if err != nil {
					aj._SetError(err)
					return
				}
				aj._Eta()
				aj._Next()
			}(job)
		}
	}()

	// Trigger first data
	max := aj.GetConcurrency()
	size := aj.jobs.Len()
	if size <= max {
		max = size
	}
	for i := 0; i < max; i++ {
		aj._Next()
	}

	go func() {
		aj.Wait()
		close(waitCh)
	}()

	var err error
	select {
	case v := <-aj.errors:
		err = v
	case <-waitCh:
		break
	}

	return err
}
