package main

import (
	"context"
	"github.com/Shopify/sarama"
	"github.com/burdiyan/kafkautil"
	"github.com/lovoo/goka"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util/kafka"
	"gitlab.com/thorchain/midgard/internal/util/miderr"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
	"strings"
)

func emitPoolEvents(ctx context.Context) chan error {

	poolEmitterGroup := goka.Group("pool-emitter")

	blockStream := goka.Stream(config.Global.Kafka.BlockTopic)
	poolStream := goka.Stream(config.Global.Kafka.PoolTopic)

	gConfig := sarama.NewConfig()
	gConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	poolEmitterG := goka.DefineGroup(poolEmitterGroup,
		goka.Input(blockStream, new(kafka.IndexedEventCodec), blockEventHandler),
		goka.Output(poolStream, new(kafka.ParsedEventCodec)),
	)

	e := make(chan error, 1)

	poolEmitProcessor, err := goka.NewProcessor(brokers, poolEmitterG,
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
		goka.WithConsumerGroupBuilder(goka.ConsumerGroupBuilderWithConfig(gConfig)),
		goka.WithHasher(kafkautil.MurmurHasher),
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

	// NOTE: we don't do any processing, we just emit events, so there is
	// no duplicate handling here. It is done by the consumers.

	event := iEvent.Event
	if event == nil {
		return
	}

	switch event.Type {
	case "errata":
		var errata record.Errata
		if err := errata.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load errata event: %v", err)
		} else {
			pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
			pE.Event = errata
			ctx.Emit(poolStream, string(errata.Asset), pE)
		}

	case "fee":
		var fee record.Fee
		if err := fee.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load fee event: %v", err)
		} else {
			pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
			pE.Event = fee
			ctx.Emit(poolStream, string(record.GetNativeAsset(fee.Asset)), pE)
		}

	case "gas":
		var gas record.Gas
		if err := gas.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load gas event: %v", err)
		} else {
			pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
			pE.Event = gas
			ctx.Emit(poolStream, string(gas.Asset), pE)
		}

	case "pool_balance_change":
		var poolBalChange record.PoolBalanceChange
		if err := poolBalChange.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load poolBalChange event: %v", err)
		} else {
			pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
			pE.Event = poolBalChange
			ctx.Emit(poolStream, string(poolBalChange.Asset), pE)
		}

	// We have to split the rewards events into one event per pool for downstream processing
	case "rewards":
		var rewards record.Rewards
		if err := rewards.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load rewards event: %v", err)
		} else {
			for _, a := range rewards.PerPool {
				pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
				amt := []record.Amount{{
					Asset: a.Asset,
					E8:    a.E8,
				}}
				pE.Event = record.Rewards{
					PerPool: amt,
				}

				ctx.Emit(poolStream, string(a.Asset), pE)
			}
		}

	case "donate":
		var donate record.Add
		if err := donate.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load donate event: %v", err)
		} else {
			pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
			pE.Event = donate
			ctx.Emit(poolStream, string(donate.Pool), pE)
		}

	case "slash":
		var slash record.Slash
		if err := slash.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load slash event: %v", err)
			return
		}
		pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
		pE.Event = slash
		ctx.Emit(poolStream, string(slash.Pool), pE)

	case "swap":
		var swap record.Swap
		if err := swap.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load swap event: %v", err)
			return
		}

		fromCoin := record.GetCoinType(swap.FromAsset)
		toCoin := record.GetCoinType(swap.ToAsset)
		if fromCoin == record.UnknownCoin {
			miderr.Printf(
				"swap event from height %d lost - unknown from Coin %s",
				iEvent.Height, swap.FromAsset)
			return
		}
		if toCoin == record.UnknownCoin {
			miderr.Printf(
				"swap event from height %d lost - unknown to Coin %s",
				iEvent.Height, swap.ToAsset)
			return
		}

		pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
		pE.Event = swap
		ctx.Emit(poolStream, string(swap.Pool), pE)

	case "add_liquidity":
		var stake record.Stake
		if err := stake.LoadTendermint(event.Attributes); err != nil {
			midlog.FatalF("Failed to load stake event: %v", err)
		} else {
			pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
			pE.Event = stake
			ctx.Emit(poolStream, string(stake.Pool), pE)
		}

	case "withdraw":
		var unstake record.Unstake
		if err := unstake.LoadTendermint(event.Attributes); err != nil {
			for _, v := range event.Attributes {
				midlog.ErrorF("%v: %v", string(v.Key), string(v.Value))
			}
			if strings.Contains(err.Error(), "nil coin") {
				midlog.ErrorF("Failed to load unstake event at height %v (t: %v): %v", iEvent.Height, iEvent.BlockTimestamp.UnixNano(), err)
			} else {
				midlog.FatalF("Failed to load unstake event at height %v (t: %v): %v", iEvent.Height, iEvent.BlockTimestamp.UnixNano(), err)
			}
			break
		}

		pE := kafka.NewParsedEventFromIndexedEvent(iEvent)
		pE.Event = unstake
		ctx.Emit(poolStream, string(unstake.Pool), pE)
	}
}
