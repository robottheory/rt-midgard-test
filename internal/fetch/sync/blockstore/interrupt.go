package blockstore

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

type InterruptSupport struct {
	signals     chan os.Signal
	interrupted int32
}

func RunWithInterruptSupport(f func(is *InterruptSupport)) {
	is := InterruptSupport{}
	is.startInterruptListener()
	f(&is)
	is.stopInterruptListener()
}

func (is *InterruptSupport) interrupt() {
	atomic.StoreInt32(&is.interrupted, 1)
}

func (is *InterruptSupport) clearInterrupted() {
	atomic.StoreInt32(&is.interrupted, 0)
}

func (is *InterruptSupport) isInterrupted() bool {
	return atomic.LoadInt32(&is.interrupted) == 1
}

func (is *InterruptSupport) startInterruptListener() {
	is.clearInterrupted()
	signals := make(chan os.Signal, 1)
	is.signals = signals
	signal.Notify(is.signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT)
	go func() {
		<-signals
		is.interrupt()
	}()
}

func (is *InterruptSupport) stopInterruptListener() {
	if is.signals != nil {
		is.signals <- syscall.SIGABRT
		is.signals = nil
	}
}
