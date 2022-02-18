[![Go](https://github.com/lab210-dev/async-job/actions/workflows/go.yml/badge.svg)](https://github.com/lab210-dev/async-job/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/lab210-dev/async-job)](https://goreportcard.com/report/github.com/lab210-dev/async-job)
[![codecov](https://codecov.io/gh/lab210-dev/async-job/branch/main/graph/badge.svg?token=3JRL5ZLSIH)](https://codecov.io/gh/lab210-dev/async-job)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/lab210-dev/async-job/blob/main/LICENSE)
# Overview

AsyncJob is an asynchronous job manager with light code, clear and speed. I hope so ! 😬

## Features

- [x] AsyncJob is a simple asynchronous job manager.
- [x] Full code coverage
- [x] Async queue
- [x] Define the number of asynchronous tasks (default: runtime.NumCPU())
- [x] Handling of managed and unmanaged errors
- [x] Provide a simple ETA
- [x] Full code description

### Usage

```go
package main

import (
	"github.com/lab210-dev/async-job"
	"log"
)

func main() {
	// Create a new AsyncJob
	asj := asyncjob.New()

	// Set the number of asynchronous tasks (default: runtime.NumCPU())
	asj.SetWorkers(2)

	// Listen to the progress status
	asj.OnProgress(func(progress asyncjob.Progress) {
            log.Printf("Progress: %s\n", progress.String())
	})

	// Run all jobs 
	err := asj.Run(func(job asyncjob.Job) error {
            // receive the job in job data function
            // if err return or panic, the job will be marked as failed and all progress will be canceled
            return nil
	}, []string{"Hello", "World"})

	// if a job returns an error, it stops the process
	if err != nil {
            log.Fatal(err)
	}
}
```

## 🤝 Contributions
Contributors to the package are encouraged to help improve the code.