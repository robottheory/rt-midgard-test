package main

import (
	"context"
	"github.com/Shopify/sarama"
	"github.com/lovoo/goka"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/util/kafka"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
)

func calculateAggregates(ctx context.Context) chan error {
	poolAggregatorGroup := goka.Group("pool-agg-stats")
	poolStatsStream := goka.Stream(config.Global.Kafka.PoolStatsTopic)

	gConfig := sarama.NewConfig()
	gConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	poolAggG := goka.DefineGroup(poolAggregatorGroup,
		goka.Input(poolStatsStream, new(kafka.IndexedEventCodec), poolStatsEventHandler),
		goka.Persist(new(pool)),
	)

	e := make(chan error, 1)

	statsProcessor, err := goka.NewProcessor(brokers, poolAggG,
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
		goka.WithConsumerGroupBuilder(goka.ConsumerGroupBuilderWithConfig(gConfig)),
	)
	if err != nil {
		e <- err
		return e
	}

	go func() {
		if err := statsProcessor.Run(ctx); err != nil {
			e <- err
		}

		close(e)
	}()

	return e
}

func poolStatsEventHandler(ctx goka.Context, msg interface{}) {
	if _, isEvent := msg.(kafka.IndexedEvent); !isEvent {
		midlog.FatalF("Processor requires value kafka.IndexedEvent, got %T", msg)
		return
	}

	var p *pool
	if val := ctx.Value(); val != nil {
		p = val.(*pool)
	} else {
		p = new(pool)
	}

	iEvent := msg.(kafka.IndexedEvent)
	if (iEvent.Height < p.lastHeight) ||
		(iEvent.Height == p.lastHeight && iEvent.Offset <= p.lastOffset) {
		// This is a duplicate event, skip it
		midlog.WarnF("Received duplicate event, height %v, offset %v", iEvent.Height, iEvent.Offset)
		return
	}

	switch ctx.Key() {
	case "add":
		p.AddCount++
		ctx.SetValue(p)
		midlog.InfoF("%v.%06v: %v, total count %v", iEvent.Height, iEvent.Offset, ctx.Key(), p.AddCount)
	case "withdraw":
		p.WithdrawCount++
		ctx.SetValue(p)
		midlog.InfoF("%v.%06v: %v, total count %v", iEvent.Height, iEvent.Offset, ctx.Key(), p.WithdrawCount)
	default:
		midlog.WarnF("Received unknown pool stats message: %v", ctx.Key())
	}
}
