package asyncjob

import (
	"fmt"
	"time"
)

// Progress is a representation of an asynchronous jobs.
type Progress struct {
	current          int
	total            int
	estimateTimeLeft time.Duration
}

// Current is the index of a current job
func (p Progress) Current() int {
	return p.current + 1
}

// Total is the sum of all jobs
func (p Progress) Total() int {
	return p.total
}

// EstimateTimeLeft is the estimated time left to finish all jobs
func (p Progress) EstimateTimeLeft() time.Duration {
	return p.estimateTimeLeft
}

// String return content of Progress as string
func (p Progress) String() string {
	return fmt.Sprintf("[%d/%d] ETA:%s", p.current, p.total, p.estimateTimeLeft)
}
