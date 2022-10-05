package main

import (
	"context"
	"github.com/Shopify/sarama"
	"github.com/burdiyan/kafkautil"
	"github.com/lovoo/goka"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util/kafka"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
)

func calculateAggregates(ctx context.Context) chan error {
	poolAggregatorGroup := goka.Group("pool-agg-stats")
	poolStatsStream := goka.Stream(config.Global.Kafka.PoolStatsTopic)

	gConfig := sarama.NewConfig()
	gConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	poolAggG := goka.DefineGroup(poolAggregatorGroup,
		goka.Input(poolStatsStream, new(kafka.ParsedEventCodec), poolStatsEventHandler),
		goka.Persist(new(pool)),
	)

	e := make(chan error, 1)

	statsProcessor, err := goka.NewProcessor(brokers, poolAggG,
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
		goka.WithConsumerGroupBuilder(goka.ConsumerGroupBuilderWithConfig(gConfig)),
		goka.WithHasher(kafkautil.MurmurHasher),
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
	if _, isEvent := msg.(kafka.ParsedEvent); !isEvent {
		midlog.FatalF("Processor requires value kafka.ParsedEvent, got %T", msg)
		return
	}

	var p *pool
	if val := ctx.Value(); val != nil {
		p = val.(*pool)
	} else {
		p = NewPool()
	}

	iEvent := msg.(kafka.ParsedEvent)

	// We are reducing partitions, so we need to track heights of the partition used in the
	// original, more partitioned, topic
	lei := p.LastEventIndexes[iEvent.OriginalPartition]

	if iEvent.EventIndex.LessOrEqual(lei) {
		// This is a duplicate event, skip it
		midlog.WarnF("AGG Received duplicate event, height %v, offset %v, partition %v",
			iEvent.EventIndex.Height, iEvent.EventIndex.Offset, iEvent.OriginalPartition)
		return
	}

	p.LastEventIndexes[iEvent.OriginalPartition] = iEvent.EventIndex

	switch ctx.Key() {
	case "rewards":
		rewards, _ := (iEvent.Event).(record.Rewards)
		p.Rewards(rewards)
		ctx.SetValue(p)

	case "slash":
		slash, _ := (iEvent.Event).(record.Slash)
		p.Slash(slash)
		ctx.SetValue(p)

	case "stake":
		stake, _ := (iEvent.Event).(record.Stake)

		_ = p.AddLiquidity(stake)
		ctx.SetValue(p)

		midlog.InfoF("%v.%06v: %v, total count %v", iEvent.EventIndex.Height, iEvent.EventIndex.Offset, ctx.Key(), p.AddCount)

	case "swap":
		swap, _ := (iEvent.Event).(record.Swap)
		p.Swap(swap)
		ctx.SetValue(p)

	case "withdraw":
		unstake, _ := (iEvent.Event).(record.Unstake)

		_ = p.WithdrawLiquidity(iEvent.EventIndex, unstake)
		ctx.SetValue(p)

		midlog.InfoF("%v.%06v: %v, total count %v", iEvent.EventIndex.Height, iEvent.EventIndex.Offset, ctx.Key(), p.WithdrawCount)
	default:
		midlog.WarnF("Received unknown pool agg message: %v", ctx.Key())
	}
}
