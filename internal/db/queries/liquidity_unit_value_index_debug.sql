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
		cast(NULL as BigInt) as withdrawn_basis_points
	from midgard.stake_events
	where pool = 'BTC.BTC' and block_timestamp < 1618330233046215552
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
			emit_asset_e8 as withdrawn_asset_e8,
			stake_units as withdrawn_stake,
			basis_points as withdrawn_basis_points
		from midgard.unstake_events
		where pool = 'BTC.BTC' and block_timestamp < 1618330233046215552
		order by block_timestamp
	)
),
-- View of stake and unstake events, together with pool depths.
stakes_and_depths as (
	select * from stake_unstake_events
	full outer join (
		select
			pool,
			asset_e8 as depth_asset_e8,
			rune_e8 as depth_rune_e8,
			block_timestamp
		from midgard.block_pool_depths
	) as depths
	using (pool, block_timestamp)
	where pool = 'BTC.BTC' and block_timestamp < 1618330233046215552
	order by block_timestamp
),
-- View of:
--   * stake and unstake event
--   * pool depths
--   * reward event entries
--   * gas events
--   * fee events
proto_block_anatomy as (
	select
        *
    from stakes_and_depths
    full outer join (
		select
			pool,
			block_timestamp,
			rune_e8 as reward_rune_e8
		from midgard.rewards_event_entries
		where pool = 'BTC.BTC' and block_timestamp < 1618330233046215552
	) as rewards
	using (pool, block_timestamp)
	full outer join (
		select
			asset as pool,
			asset_e8 as gas_event_asset_e8,
			rune_e8 as gas_event_rune_e8,
			block_timestamp
		from midgard.gas_events
		where asset = 'BTC.BTC' and block_timestamp < 1618330233046215552
	) as gas
	using (pool, block_timestamp)
	full outer join (
		select
			asset as pool,
			asset_e8 as fee_event_asset_e8,
			pool_deduct as fee_event_rune_e8,
			block_timestamp
		from midgard.fee_events
		where asset = 'BTC.BTC' and block_timestamp < 1618330233046215552
	) as fee
	using (pool, block_timestamp)
),
block_anatomy as (
	select
		*,
		sum(coalesce(added_stake, 0)) over wnd - sum(coalesce(withdrawn_stake, 0)) over wnd as total_stake,
		last_value(depth_asset_e8::numeric) over wnd * last_value(depth_rune_e8) over wnd as depth_product
	from proto_block_anatomy
	window wnd as (partition by pool order by block_timestamp nulls last)
	order by block_timestamp
)
select * from (
select
	*,
	sum(depth_asset_e8 - prev_depth_asset_e8
		+ withdrawn_asset_e8 - added_asset_e8
		+ gas_event_asset_e8 - fee_event_asset_e8) over (partition by block_timestamp, pool) as asset_chg_check,
	depth_rune_e8 - prev_depth_rune_e8
		+ withdrawn_rune_e8 - added_rune_e8
		+ fee_event_rune_e8 - gas_event_rune_e8 - reward_rune_e8 as rune_chg_check
from (
	select
		pool,
		block_timestamp,
		to_timestamp(block_timestamp / 1000000000)::date as date,
		member_addr,
		asset_addr,
		coalesce(added_rune_e8, 0) as added_rune_e8,
		coalesce(added_asset_e8, 0) as added_asset_e8,
		coalesce(added_stake, 0) as added_stake,
		coalesce(withdrawn_rune_e8, 0) as withdrawn_rune_e8,
		coalesce(withdrawn_asset_e8, 0) as withdrawn_asset_e8,
		coalesce(withdrawn_stake, 0) as withdrawn_stake,
		coalesce(reward_rune_e8, 0) as reward_rune_e8,
		coalesce(gas_event_asset_e8, 0) as gas_event_asset_e8,
		coalesce(gas_event_rune_e8, 0) as gas_event_rune_e8,
		coalesce(fee_event_asset_e8, 0) as fee_event_asset_e8,
		coalesce(fee_event_rune_e8, 0) as fee_event_rune_e8,
		total_stake,
		depth_asset_e8,
		depth_rune_e8,
		lag(total_stake, 1) over (partition by pool order by block_timestamp) as prev_total_stake,
		lag(depth_asset_e8, 1) over (partition by pool order by block_timestamp) as prev_depth_asset_e8,
		lag(depth_rune_e8, 1) over (partition by pool order by block_timestamp) as prev_depth_rune_e8,
		sqrt(depth_product) / total_stake as liquidity_unit_value_index,
		sqrt(depth_product) / total_stake < lag(sqrt(depth_product) / total_stake, 1)
			over (partition by pool order by block_timestamp) as decrease,
		sqrt(depth_product) / total_stake
			/ sqrt(lag(depth_product / total_stake / total_stake, 1)
			over (partition by pool order by block_timestamp)) - 1 as pct_change
	from block_anatomy
	where pool = 'BTC.BTC'
) as anatomy
) as anatomy_with_check
where decrease = true and pool = 'BTC.BTC'
limit 50	
    
