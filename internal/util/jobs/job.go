package jobs

import (
	"context"
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

func (q *Job) Wait(finishCTX context.Context) {
	if q == nil || q.quitFinished == nil {
		return
	}
	log.Printf("Waiting %s goroutine to finish.\n", q.name)
	select {
	case <-q.quitFinished:
		log.Printf("%s stopped.", q.name)
		return
	default:
	}

	select {
	case <-q.quitFinished:
		log.Printf("%s stopped.", q.name)
		return
	case <-finishCTX.Done():
		log.Printf("Failed to stop %s goroutine within timeout.", q.name)
		return
	}
}

func (q *Job) MustWait() {
	<-q.quitFinished
}
