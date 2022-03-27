package asyncjob

// Job is a representation of a unitary work
type Job[T any] struct {
	index int
	data  T
}

// Data is the value of a job
func (j Job[T]) Data() T {
	return j.data
}

// Index the index of the original slice
func (j Job[T]) Index() int {
	return j.index
}
