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

func emitPoolEvents(ctx context.Context) chan error {
	poolEmitterGroup := goka.Group("pool-emitter")

	blockStream := goka.Stream(config.Global.Kafka.BlockTopic)
	poolStream := goka.Stream(config.Global.Kafka.PoolTopic)

	gConfig := sarama.NewConfig()
	gConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	poolEmitterG := goka.DefineGroup(poolEmitterGroup,
		goka.Input(blockStream, new(kafka.IndexedEventCodec), blockEventHandler),
		goka.Output(poolStream, new(kafka.IndexedEventCodec)),
	)

	e := make(chan error, 1)

	poolEmitProcessor, err := goka.NewProcessor(brokers, poolEmitterG,
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
		goka.WithConsumerGroupBuilder(goka.ConsumerGroupBuilderWithConfig(gConfig)),
	)
	if err != nil {
		e <- err
		return e
	}

	go func() {
		if err := poolEmitProcessor.Run(ctx); err != nil {
			e <- err
		}
	}()

	return e
}

// This callback handles messages from the blocks topic and sends them to the pool topic
func blockEventHandler(ctx goka.Context, msg interface{}) {
	poolStream := goka.Stream(config.Global.Kafka.PoolTopic)

	if _, isEvent := msg.(kafka.IndexedEvent); !isEvent {
		midlog.FatalF("Processor requires value kafka.IndexedEvent, got %T", msg)
		return
	}

	iEvent := msg.(kafka.IndexedEvent)

	event := iEvent.Event
	bh := iEvent.BlockTimestamp.UnixNano()
	bh = bh + 1
	switch event.Type {
	case "add_liquidity":
		var stake record.Stake
		if err := stake.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load stake event: %v", err)
		} else {
			ctx.Emit(poolStream, string(stake.Pool), iEvent)
		}

	case "withdraw":
		var unstake record.Unstake

		meta := record.Metadata{
			BlockHeight:    iEvent.Height,
			BlockTimestamp: iEvent.BlockTimestamp,
		}

		if record.CorrectWithdaw(&unstake, &meta) == record.Discard {
			break
		}

		if err := unstake.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load unstake event: %v", err)
		} else {
			ctx.Emit(poolStream, string(unstake.Pool), iEvent)
		}

	}
}
