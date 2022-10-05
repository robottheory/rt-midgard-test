package main

import (
	"context"
	"database/sql"
	"github.com/Shopify/sarama"
	"github.com/burdiyan/kafkautil"
	"github.com/lovoo/goka"
	"gitlab.com/thorchain/midgard/config"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util/kafka"
	"gitlab.com/thorchain/midgard/internal/util/midlog"
)

var db *sql.DB

func emitPoolStatsEvents(ctx context.Context) chan error {
	connStr := "postgresql://pool:pool@localhost/pools?sslmode=disable"
	// Connect to database
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	poolStatsGroup := goka.Group("pool-stats")
	poolStatsStream := goka.Stream(config.Global.Kafka.PoolStatsTopic)
	poolStream := goka.Stream(config.Global.Kafka.PoolTopic)

	gConfig := sarama.NewConfig()
	gConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	g := goka.DefineGroup(poolStatsGroup,
		goka.Input(poolStream, new(kafka.ParsedEventCodec), poolEventHandler),
		goka.Output(poolStatsStream, new(kafka.ParsedEventCodec)),
		goka.Persist(new(pool)),
	)

	e := make(chan error, 1)

	poolProcessor, err := goka.NewProcessor(brokers, g,
		goka.WithTopicManagerBuilder(goka.TopicManagerBuilderWithTopicManagerConfig(tmc)),
		goka.WithConsumerGroupBuilder(goka.ConsumerGroupBuilderWithConfig(gConfig)),
		goka.WithHasher(kafkautil.MurmurHasher),
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

	if _, isEvent := msg.(kafka.ParsedEvent); !isEvent {
		midlog.FatalF("Processor requires value kafka.ParsedEvent, got %T", msg)
		return
	}

	iEvent := msg.(kafka.ParsedEvent)

	// Note this will always be nil for event types we don't handle
	var p *pool
	if val := ctx.Value(); val != nil {
		p = val.(*pool)
	} else {
		p = NewPool()
	}

	lei := p.LastEventIndexes[ctx.Partition()]

	if iEvent.EventIndex.LessOrEqual(lei) {
		// This is a duplicate event, skip it
		midlog.WarnF("Received duplicate event, height %v, offset %v",
			iEvent.EventIndex.Height, iEvent.EventIndex.Offset)
		return
	}

	p.LastEventIndexes[ctx.Partition()] = iEvent.EventIndex

	// Can be used by receivers to detect duplicate events because
	// the target topic has fewer partitions than the source topic
	iEvent.OriginalPartition = ctx.Partition()

	// All events that reach this point will have been validated by the prior emitter
	switch iEvent.Type {
	case "errata":
		errata, _ := (iEvent.Event).(record.Errata)
		p.Errata(errata)
		ctx.SetValue(p)

	case "fee":
		fee, _ := (iEvent.Event).(record.Fee)
		p.Fee(fee)
		ctx.SetValue(p)

	case "gas":
		gas, _ := (iEvent.Event).(record.Gas)
		p.Gas(gas)
		ctx.SetValue(p)

	case "pool_balance_change":
		poolBalChange, _ := (iEvent.Event).(record.PoolBalanceChange)
		p.PoolBalChange(poolBalChange)
		ctx.SetValue(p)

	case "rewards":
		rewards, _ := (iEvent.Event).(record.Rewards)
		p.Rewards(rewards)
		ctx.SetValue(p)

	case "donate":
		donate, _ := (iEvent.Event).(record.Add)
		p.Donate(donate)
		ctx.SetValue(p)

		ctx.Emit(poolStatsStream, "donate", iEvent)

	case "slash":
		slash, _ := (iEvent.Event).(record.Slash)
		p.Slash(slash)
		ctx.SetValue(p)

	case "swap":
		swap, _ := (iEvent.Event).(record.Swap)
		p.Swap(swap)
		ctx.SetValue(p)

		ctx.Emit(poolStatsStream, "swap", iEvent)

	case "add_liquidity":
		stake, _ := (iEvent.Event).(record.Stake)

		assetInRune := p.AddLiquidity(stake)
		ctx.SetValue(p)

		ctx.Emit(poolStatsStream, "stake", iEvent)

		q := "INSERT INTO stake_events (pool, asset_tx, asset_chain, asset_addr, asset_e8, stake_units, rune_tx, rune_addr, rune_e8, _asset_in_rune_e8, block_timestamp) " +
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)"
		_, err := db.Exec(q, stake.Pool, stake.AssetTx, stake.AssetChain, stake.AssetAddr, stake.AssetE8, stake.StakeUnits,
			stake.RuneTx, stake.RuneAddr, stake.RuneE8, assetInRune, iEvent.BlockTimestamp.UnixNano())
		//exec.RowsAffected()
		if err != nil {
			midlog.WarnF("Err: %v", err)
		}

		//midlog.InfoF("%v.%v: %v, %v count %v", iEvent.Height, iEvent.Offset, ctx.Key(), event.Type, p.AddCount)
	case "withdraw":
		unstake, ok := (iEvent.Event).(record.Unstake)
		if !ok {
			midlog.FatalF("Wrong type, got: %T", iEvent.Event)
		}

		assetInRune := p.WithdrawLiquidity(iEvent.EventIndex, unstake)

		ctx.SetValue(p)

		ctx.Emit(poolStatsStream, "withdraw", iEvent)

		q := "INSERT INTO unstake_events (tx, chain, from_addr, to_addr, asset, asset_e8, emit_asset_e8, emit_rune_e8, memo, pool, stake_units, basis_points, asymmetry, imp_loss_protection_e8, _emit_asset_in_rune_e8, block_timestamp, height, offset, partition) " +
			"VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)"
		_, err := db.Exec(q, unstake.Tx, unstake.Chain, unstake.FromAddr, unstake.ToAddr, unstake.Asset,
			unstake.AssetE8, unstake.EmitAssetE8, unstake.EmitRuneE8, unstake.Memo, unstake.Pool, unstake.StakeUnits,
			unstake.BasisPoints, unstake.Asymmetry, unstake.ImpLossProtectionE8, assetInRune, iEvent.BlockTimestamp.UnixNano(),
			iEvent.EventIndex.Height, iEvent.EventIndex.Offset, ctx.Partition())
		//exec.RowsAffected()
		if err != nil {
			midlog.WarnF("Err: %v", err)
		}

		//midlog.InfoF("%v.%d: %v, %v count %v", iEvent.Height, iEvent.Offset, ctx.Key(), event.Type, p.WithdrawCount)

	default:
		midlog.WarnF("Received unknown pool stats message: %v", ctx.Key())
	}

}
