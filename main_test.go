package asyncjob

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"testing"
	"time"
)

// function for print log with testing instance
func LogJob[T any](t *testing.T, job Job[T]) {
	t.Logf("OnJob=%T key=%d", job.Data(), job.Index())
}

func TestNew(t *testing.T) {
	t.Run("Run with empty slice data", func(t *testing.T) {
		err := New[string]().
			Run(func(job Job[string]) error {
				return nil
			}, []string{})
		if assert.Nil(t, err) {
			return
		}
	})
	t.Run("Run with valid listener", func(t *testing.T) {
		var m sync.Mutex
		var found = make([]string, 2)
		var list = []string{"Hello", "World"}
		err := New[string]().
			Run(func(job Job[string]) error {
				m.Lock()
				defer m.Unlock()
				LogJob(t, job)
				found[job.Index()] = job.Data()
				return nil
			}, list)

		if !assert.Nil(t, err) {
			return
		}
		t.Log(found)
	})
	t.Run("Run with job panic", func(t *testing.T) {
		var m sync.Mutex
		var found = make([]string, 2)
		var list = []string{"Hello", "World"}
		err := New[string]().Run(func(job Job[string]) error {
			m.Lock()
			defer m.Unlock()
			LogJob(t, job)
			found[job.Index()] = job.Data()
			panic("i'm panic")
			return nil
		}, list)
		if !assert.NotNil(t, err) {
			return
		}
		assert.Equal(t, "i'm panic", err.Error())
	})
	t.Run("Run with job return error", func(t *testing.T) {
		var m sync.Mutex
		var found = make([]string, 2)
		var list = []string{"Hello", "World"}
		err := New[string]().Run(func(job Job[string]) error {
			m.Lock()
			defer m.Unlock()
			LogJob(t, job)
			found[job.Index()] = job.Data()
			return errors.New("i'm a error")
		}, list)
		if !assert.NotNil(t, err) {
			return
		}
		assert.Equal(t, "i'm a error", err.Error())
	})
	t.Run("Run with large data", func(t *testing.T) {
		var m sync.Mutex
		var found = make([]int, 1000)
		var list []int
		for i := 1; i <= 1000; i++ {
			list = append(list, i)
		}

		err := New[int]().
			OnProgress(func(progress Progress) {}).
			Run(func(job Job[int]) error {
				m.Lock()
				defer m.Unlock()
				found[job.Index()] = job.Data()
				return nil
			}, list)

		if !assert.Nil(t, err) {
			return
		}
		// test if found is equal to list
		assert.Equal(t, len(list), len(found))
	})
	t.Run("Run with basic slice with sleep", func(t *testing.T) {
		var m sync.Mutex
		var found = make([]int, 50)
		var list []int
		var d time.Duration
		for i := 1; i <= 50; i++ {
			list = append(list, i)
			d = d + time.Duration(i)*time.Millisecond
		}
		startTimer := time.Now()
		err := New[int]().Run(func(job Job[int]) error {
			m.Lock()
			defer m.Unlock()
			found[job.Index()] = job.Data()
			time.Sleep(time.Duration(job.Index()) * time.Millisecond)
			return nil
		}, list)

		// test err if nil
		if !assert.Nil(t, err) {
			return
		}

		// end timer
		endTimer := time.Now()

		// calcul percentage of timer
		percentage := float64(endTimer.Sub(startTimer)) / float64(d)
		t.Logf("slowness margin (>5%%)= %.2f%% (%s)", percentage, endTimer.Sub(startTimer))

		// test if percentage is greater than 95%
		assert.True(t, percentage < 5)

		// test if found is equal to list
		assert.Equal(t, len(list), len(found))
	})
	t.Run("Reliability testing ", func(t *testing.T) {
		var list []time.Duration
		var final []Job[time.Duration]
		var m sync.Mutex
		for i := 1; i <= 4; i++ {
			if i == 1 {
				list = append(list, time.Duration(1)*time.Second)
				continue
			}
			if i == 3 {
				list = append(list, time.Duration(2)*time.Second)
				continue
			}
			list = append(list, time.Duration(0)*time.Second)
		}
		log.Print(list)
		startTimer := time.Now()
		err := New[time.Duration]().SetWorkers(2).Run(func(job Job[time.Duration]) error {
			m.Lock()
			t.Logf("Starting Index:%d Time:%f D:%d", job.Index(), time.Since(startTimer).Seconds(), job.Data())
			final = append(final, job)
			m.Unlock()
			time.Sleep(job.Data())
			t.Logf("Ending Index:%d Time:%f D:%d", job.Index(), time.Since(startTimer).Seconds(), job.Data())
			m.Lock()
			final = append(final, job)
			m.Unlock()
			return nil
		}, list)

		// test err if nil
		if !assert.Nil(t, err) {
			return
		}

		assert.True(t, time.Since(startTimer).Seconds() >= 2)
		log.Print(final)

		assert.Equal(t, final[0].Data(), time.Duration(1)*time.Second)
		assert.Equal(t, final[0].Index(), 0)

		assert.Equal(t, final[1].Index(), 1)
		assert.Equal(t, final[1].Data(), time.Duration(0)*time.Second)

		assert.Equal(t, final[2].Index(), 1)
		assert.Equal(t, final[2].Data(), time.Duration(0)*time.Second)

		assert.Equal(t, final[3].Index(), 2)
		assert.Equal(t, final[3].Data(), time.Duration(2)*time.Second)

		assert.Equal(t, final[4].Index(), 0)
		assert.Equal(t, final[4].Data(), time.Duration(1)*time.Second)

		assert.Equal(t, final[5].Index(), 3)
		assert.Equal(t, final[5].Data(), time.Duration(0)*time.Second)

		assert.Equal(t, final[6].Index(), 3)
		assert.Equal(t, final[6].Data(), time.Duration(0)*time.Second)

		assert.Equal(t, final[7].Index(), 2)
		assert.Equal(t, final[7].Data(), time.Duration(2)*time.Second)
	})
	t.Run("Testing eta", func(t *testing.T) {
		var list []time.Duration
		for i := 1; i <= 5; i++ {
			list = append(list, time.Duration(1)*time.Second)
		}
		var result []float64
		err := New[time.Duration]().
			SetWorkers(2).
			OnProgress(func(progress Progress) {
				t.Log(progress.Current(), progress.Total(), progress.EstimateTimeLeft(), progress)
				result = append(result, progress.EstimateTimeLeft().Seconds())
			}).
			Run(func(job Job[time.Duration]) error {
				time.Sleep(job.Data())
				return nil
			}, list)

		// test err if nil
		if !assert.Nil(t, err) {
			return
		}
		assert.True(t, result[0] >= 1.5 && result[0] <= 1.6)
		assert.True(t, result[1] >= 0.600 && result[1] <= 0.700)
		assert.True(t, result[2] >= 0.500 && result[2] <= 0.600)
		t.Log(result)
	})
	t.Run("Testing large eta", func(t *testing.T) {
		var list []time.Duration
		for i := 1; i <= 5000; i++ {
			list = append(list, time.Duration(1)*time.Microsecond)
		}
		var result []float64
		err := New[time.Duration]().
			SetWorkers(2).
			OnProgress(func(progress Progress) {
				if progress.Current()%1000 != 0 {
					return
				}
				t.Log(progress.Current(), progress.Total(), progress.EstimateTimeLeft(), progress)
				result = append(result, progress.EstimateTimeLeft().Seconds())
			}).
			Run(func(job Job[time.Duration]) error {
				time.Sleep(job.Data())
				return nil
			}, list)

		// test err if nil
		if !assert.Nil(t, err) {
			return
		}
		// verify all value into result
		for _, v := range result {
			assert.True(t, v != 0)
		}
		t.Log(result)
	})
}
