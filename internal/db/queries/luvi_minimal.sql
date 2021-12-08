-- View of the union of stake and unstake events.
with stake_unstake_events as (
	select
		pool,
		block_timestamp,
		coalesce(rune_addr, asset_addr) as member_addr,
		asset_addr,
		rune_e8 as added_rune_e8,
		asset_e8 as added_asset_e8,
		stake_units as added_stake,
		cast(NULL as BigInt) as withdrawn_rune_e8,
		cast(NULL as BigInt) as withdrawn_asset_e8,
		cast(NULL as BigInt) as withdrawn_stake,
		cast(NULL as BigInt) as imp_loss_protection_e8,
		cast(NULL as BigInt) as withdrawn_basis_points
	from midgard.stake_events
	--where pool = 'BTC.BTC'
	union (
		select
			pool,
			block_timestamp,
			from_addr as member_addr,
			cast(NULL as text) as asset_addr,
			cast(NULL as BigInt) as added_rune_e8,
			cast(NULL as BigInt) as added_asset_e8,
			cast(NULL as BigInt) as added_stake,
			emit_rune_e8 as withdrawn_rune_e8,
			emit_asset_e8
				+ case when asset = pool and block_timestamp < 1630938167525799522
					then -asset_e8 else 0 end as withdrawn_asset_e8,
			stake_units as withdrawn_stake,
			imp_loss_protection_e8,
			basis_points as withdrawn_basis_points
		from midgard.unstake_events
		--where pool = 'BTC.BTC'
		order by block_timestamp
	)
),
-- Aggregated liquidity events by pool and block_timestamp.
liquidity_events_summary as (
	select
		pool,
		block_timestamp,
		sum(added_rune_e8) as added_rune_e8,
		sum(added_asset_e8) as added_asset_e8,
		sum(added_stake) as added_stake,
		sum(withdrawn_rune_e8) as withdrawn_rune_e8,
		sum(withdrawn_asset_e8) as withdrawn_asset_e8,
		sum(withdrawn_stake) as withdrawn_stake,
		sum(imp_loss_protection_e8) as imp_loss_protection_e8
    from stake_unstake_events
    group by pool, block_timestamp
),
-- Block summary including:
--	* pool depths
--	* aggregate stake and unstake events
block_summary as (
	select
		*
	from liquidity_events_summary
	full outer join (
		select
			pool,
			asset_e8 as depth_asset_e8,
			rune_e8 as depth_rune_e8,
			block_timestamp
		from midgard.block_pool_depths
        where block_timestamp >= 1638924547138005938
		--where pool = 'BTC.BTC'
	) as depths
	using (pool, block_timestamp)
    where block_timestamp >= 1638924547138005938
),

-- Summary of events per block together with total stake.
blocks as (
	select
		pool,
		block_timestamp,
		to_timestamp(block_timestamp / 1000000000)::date as date,
		depth_asset_e8,
		depth_rune_e8,
		sum(coalesce(added_stake, 0)) over cumulative_wnd
			- sum(coalesce(withdrawn_stake, 0)) over cumulative_wnd as total_stake
	from block_summary
	window cumulative_wnd as (partition by pool order by block_timestamp)
),

proto_metrics as (
	select
		*,
		depth_asset_e8::numeric * depth_rune_e8 as depth_product,
		(depth_asset_e8::numeric + 1) * (depth_rune_e8 + 1) as depth_product1
	from blocks
	order by block_timestamp
),
metrics as (
	select
		pool,
		block_timestamp,
		to_timestamp(block_timestamp / 1000000000)::date as date,
		total_stake,
		depth_asset_e8,
		depth_rune_e8,
		lag(total_stake, 1) over wnd as prev_total_stake,
		lag(depth_asset_e8, 1) over wnd as prev_depth_asset_e8,
		lag(depth_rune_e8, 1) over wnd as prev_depth_rune_e8,
		sqrt(depth_product) / total_stake as luvi,
		lag(sqrt(depth_product) / total_stake, 1) over wnd as prev_luvi,
		sqrt(depth_product) / total_stake < lag(sqrt(depth_product) / total_stake, 1) over wnd as luvi_decrease,
		sqrt(depth_product1) / total_stake < lag(sqrt(depth_product) / total_stake, 1) over wnd as luvi_decrease1,
		sqrt(depth_product) / total_stake
			/ lag(sqrt(depth_product) / total_stake, 1) over wnd - 1 as pct_change,
		sqrt(depth_product1) / total_stake
			/ lag(sqrt(depth_product) / total_stake, 1) over wnd - 1 as pct_change1
	from proto_metrics
    where total_stake != 0
	--where pool = 'BTC.BTC'
	window wnd as (partition by pool order by block_timestamp)
),
daily_metrics as (
	select
		*
	from (
		select
			pool,
			block_timestamp,
			date,
			luvi,
			depth_asset_e8,
			depth_rune_e8,
			total_stake,
			row_number() over (partition by pool, block_timestamp / 1000000000 / 60 / 60 / 24
							   order by block_timestamp desc) as r
		from metrics
		where depth_rune_e8 >= 0
		window wnd as (partition by pool, date)
	) as sequenced
	where r = 1
),
weekly_metrics as (
	select * from (
		select
			pool,
			block_timestamp,
			date,
			luvi,
			depth_asset_e8,
			depth_rune_e8,
			total_stake,
			row_number() over (partition by pool, block_timestamp / 1000000000 / 60 / 60 / 24 / 7
							order by block_timestamp desc) as r
		from metrics
		where depth_rune_e8 >= 0
		window wnd as (partition by pool, date)
	) as sequenced
	where r = 1
)
select * from metrics where luvi_decrease1 = true