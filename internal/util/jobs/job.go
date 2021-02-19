package jobs

import (
	"context"
	"fmt"
	"log"
)

type Job struct {
	quitFinished chan struct{}
	name         string
}

func Start(name string, job func()) Job {
	ret := Job{quitFinished: make(chan struct{}), name: name}
	go func() {
		job()
		ret.quitFinished <- struct{}{}
	}()
	return ret
}

func (q *Job) Wait(finishCTX context.Context) error {
	if q == nil || q.quitFinished == nil {
		return nil
	}
	log.Printf("Waiting %s goroutine to finish.\n", q.name)
	select {
	case <-q.quitFinished:
		log.Printf("%s stopped.", q.name)
		return nil
	default:
	}

	select {
	case <-q.quitFinished:
		log.Printf("%s stopped.", q.name)
		return nil
	case <-finishCTX.Done():
		log.Printf("Failed to stop %s goroutine within timeout.", q.name)
		return fmt.Errorf("Failed to stop %s goroutine within timeout.", q.name)
	}
}

func WaitAll(finishCTX context.Context, allJobs ...*Job) {
	allErrors := []error{}
	for _, job := range allJobs {
		err := job.Wait(finishCTX)
		if err != nil {
			allErrors = append(allErrors, err)
		}
	}
	for _, err := range allErrors {
		log.Println(err)
	}
}

func (q *Job) MustWait() {
	<-q.quitFinished
}
