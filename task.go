package asyncjob

// Job is a representation of a unitary work
type Job struct {
	index int
	data  interface{}
}

// Data is the value of a job
func (j Job) Data() interface{} {
	return j.data
}

// Index the index of the original slice
func (j Job) Index() int {
	return j.index
}
