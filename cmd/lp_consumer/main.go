package main

import (
	"context"
	"github.com/lovoo/goka"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
	"os"
	"os/signal"
	"syscall"
)

var (
	tmc *goka.TopicManagerConfig

	brokers []string
)

func init() {
	tmc = goka.NewTopicManagerConfig()
	tmc.Table.Replication = 1
	tmc.Stream.Replication = 1
}

func main() {
	mainCtx, done := context.WithCancel(context.Background())

	midlog.LogCommandLine()
	config.ReadGlobal()

	brokers = config.Global.Kafka.Brokers

	midlog.Info("Starting pool emitter")
	pe := emitPoolEvents(mainCtx)

	midlog.Info("Starting pool stats emitter/calculator")
	ps := emitPoolStatsEvents(mainCtx)

	midlog.Info("Starting aggregate calculator")
	pa := calculateAggregates(mainCtx)

	sigs := make(chan os.Signal)
	go func() {
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	}()

	select {
	case e := <-pe:
		midlog.FatalE(e, "Error running pool emitter")
	case e := <-ps:
		midlog.FatalE(e, "Error running stats emitter")
	case e := <-pa:
		midlog.FatalE(e, "Error running aggregate calculator")
	case <-sigs:
		midlog.Info("Signal received, exiting")
	}

	done()

}
