package main

import (
	"context"
	"github.com/Shopify/sarama"
	"github.com/lovoo/goka"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util/kafka"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
)

func emitPoolStatsEvents(ctx context.Context) chan error {
	poolStatsGroup := goka.Group("pool-stats")
	poolStatsStream := goka.Stream(config.Global.Kafka.PoolStatsTopic)
	poolStream := goka.Stream(config.Global.Kafka.PoolTopic)

	gConfig := sarama.NewConfig()
	gConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	g := goka.DefineGroup(poolStatsGroup,
		goka.Input(poolStream, new(kafka.IndexedEventCodec), poolEventHandler),
		goka.Output(poolStatsStream, new(kafka.IndexedEventCodec)),
		goka.Persist(new(pool)),
	)

	e := make(chan error, 1)

	poolProcessor, err := goka.NewProcessor(brokers, g,
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
		goka.WithConsumerGroupBuilder(goka.ConsumerGroupBuilderWithConfig(gConfig)),
	)
	if err != nil {
		e <- err
		return e
	}

	go func() {
		if err := poolProcessor.Run(ctx); err != nil {
			e <- err
		}
	}()

	return e
}

func poolEventHandler(ctx goka.Context, msg interface{}) {
	poolStatsStream := goka.Stream(config.Global.Kafka.PoolStatsTopic)

	if _, isEvent := msg.(kafka.IndexedEvent); !isEvent {
		midlog.ErrorF("Processor requires value kafka.IndexedEvent, got %T", msg)
		return
	}

	iEvent := msg.(kafka.IndexedEvent)
	event := iEvent.Event

	// Note this will always be nil for event types we don't handle
	var p *pool
	if val := ctx.Value(); val != nil {
		p = val.(*pool)
	} else {
		p = new(pool)
	}

	if (iEvent.Height < p.lastHeight) ||
		(iEvent.Height == p.lastHeight && iEvent.Offset <= p.lastOffset) {
		// This is a duplicate event, skip it
		midlog.WarnF("Received duplicate event, height %v, offset %v", iEvent.Height, iEvent.Offset)
		return
	}

	p.lastHeight = iEvent.Height
	p.lastOffset = iEvent.Offset

	switch event.Type {
	case "add_liquidity":
		var stake record.Stake
		stake.LoadTendermint(event.Attributes)

		p.AddCount++
		ctx.SetValue(p)

		ctx.Emit(poolStatsStream, "add", iEvent)

		midlog.InfoF("%v.%v: %v, %v add count %v", iEvent.Height, iEvent.Offset, ctx.Key(), event.Type, p.AddCount)
	case "withdraw":
		var stake record.Unstake
		stake.LoadTendermint(event.Attributes)

		p.WithdrawCount++
		ctx.SetValue(p)

		ctx.Emit(poolStatsStream, "withdraw", iEvent)

		midlog.InfoF("%v.%d: %v, %v add count %v", iEvent.Height, iEvent.Offset, ctx.Key(), event.Type, p.WithdrawCount)
	}

}
