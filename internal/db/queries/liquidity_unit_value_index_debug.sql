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
	where pool = 'BTC.BTC'
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
		where pool = 'BTC.BTC'
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
	where pool = 'BTC.BTC'
	order by block_timestamp
),
-- View of:
--   * pool depths
--   * stake and unstake event
--   * reward events
--   * gas events
--   * fee events
events_and_depths as (
	select
        *
    from stakes_and_depths
    full outer join (
		select
			pool,
			block_timestamp,
			rune_e8 as reward_rune_e8
		from midgard.rewards_event_entries
		where pool = 'BTC.BTC'
	) as rewards
	using (pool, block_timestamp)
	full outer join (
		select
			asset as pool,
			asset_e8 as gas_event_asset_e8,
			rune_e8 as gas_event_rune_e8,
			block_timestamp
		from midgard.gas_events
		where asset = 'BTC.BTC'
	) as gas
	using (pool, block_timestamp)
	full outer join (
		select
			asset as pool,
			asset_e8 as fee_event_asset_e8,
			pool_deduct as fee_event_rune_e8,
			block_timestamp
		from midgard.fee_events
		where asset = 'BTC.BTC'
	) as fee
	using (pool, block_timestamp)
),
-- Summary of events per block.
blocks as (
	select distinct on (pool, block_timestamp)
		last_value(pool) over block_wnd as pool,
		last_value(block_timestamp) over block_wnd as block_timestamp,
		sum(added_asset_e8) over block_wnd as A_a_e8,
		sum(added_rune_e8) over block_wnd as A_r_e8,
		sum(added_stake) over block_wnd as A_s,
		sum(withdrawn_asset_e8) over block_wnd as W_a_e8,
		sum(withdrawn_rune_e8) over block_wnd as W_r_e8,
		sum(withdrawn_stake) over block_wnd as W_s,
		sum(fee_event_asset_e8) over block_wnd as F_a_e8,
		sum(fee_event_rune_e8) over block_wnd as F_r_e8,
		sum(gas_event_asset_e8) over block_wnd as G_a_e8,
		sum(gas_event_rune_e8) over block_wnd as G_r_e8,
		sum(reward_rune_e8) over block_wnd as R_r_e8,
		last_value(depth_asset_e8) over block_wnd as depth_asset_e8,
		last_value(depth_rune_e8) over block_wnd as depth_rune_e8,
		sum(coalesce(added_stake, 0)) over cumulative_wnd
			- sum(coalesce(withdrawn_stake, 0)) over cumulative_wnd as total_stake,
		last_value(depth_asset_e8::numeric * depth_rune_e8) over block_wnd as depth_product,
		last_value((depth_asset_e8::numeric + 1) * (depth_rune_e8 + 1)) over block_wnd as depth_product1,
		last_value((depth_asset_e8::numeric + 1) *
			(depth_rune_e8
			- gas_event_rune_e8
				+ (gas_event_asset_e8::numeric * depth_rune_e8
					/ (depth_asset_e8 - gas_event_asset_e8))
			- fee_event_rune_e8
			+ (fee_event_asset_e8::numeric * depth_rune_e8
				/ (depth_asset_e8 - fee_event_asset_e8)) + 1)) over block_wnd as depth_product2
	from events_and_depths
	window cumulative_wnd as (partition by pool order by block_timestamp nulls last),
		block_wnd as (partition by pool, block_timestamp)
	order by block_timestamp
)
select * from (
select
	*,
	depth_asset_e8 - prev_depth_asset_e8
		+ withdrawn_asset_e8 - added_asset_e8
		+ gas_event_asset_e8 - fee_event_asset_e8 as asset_chg_check,
	depth_rune_e8 - prev_depth_rune_e8
		+ withdrawn_rune_e8 - added_rune_e8
		+ fee_event_rune_e8 - gas_event_rune_e8 - reward_rune_e8 as rune_chg_check
from (
	select
		pool,
		block_timestamp,
		to_timestamp(block_timestamp / 1000000000)::date as date,
		coalesce(A_a_e8, 0) as added_rune_e8,
		coalesce(A_r_e8, 0) as added_asset_e8,
		coalesce(A_s, 0) as added_stake,
		coalesce(W_r_e8, 0) as withdrawn_rune_e8,
		coalesce(W_a_e8, 0) as withdrawn_asset_e8,
		coalesce(W_s, 0) as withdrawn_stake,
		coalesce(R_r_e8, 0) as reward_rune_e8,
		coalesce(G_a_e8, 0) as gas_event_asset_e8,
		coalesce(G_r_e8, 0) as gas_event_rune_e8,
		coalesce(F_a_e8, 0) as fee_event_asset_e8,
		coalesce(F_r_e8, 0) as fee_event_rune_e8,
		total_stake,
		depth_asset_e8,
		depth_rune_e8,
		lag(total_stake, 1) over wnd as prev_total_stake,
		lag(depth_asset_e8, 1) over wnd as prev_depth_asset_e8,
		lag(depth_rune_e8, 1) over wnd as prev_depth_rune_e8,
		sqrt(depth_product) / total_stake as liquidity_unit_value_index,
		lag(sqrt(depth_product) / total_stake, 1) over wnd as prev_liquidity_unit_value_index,
		sqrt(depth_product) / total_stake < lag(sqrt(depth_product) / total_stake, 1) over wnd as decrease,
		sqrt(depth_product1) / total_stake < lag(sqrt(depth_product) / total_stake, 1) over wnd as decrease1,
		sqrt(depth_product2) / total_stake < lag(sqrt(depth_product) / total_stake, 1) over wnd as decrease2,
		sqrt(depth_product) / total_stake
			/ sqrt(lag(depth_product / total_stake / total_stake, 1) over wnd) - 1 as pct_change
	from blocks
	where pool = 'BTC.BTC'
	window wnd as (partition by pool order by block_timestamp)
) as metrics
) as metrics_with_checks

GAS EVENTS NEED TO BE AGGREGATED BEFORE USED IN THE ADJustED PRODUcT DEPTH FORMULA
where decrease2 = true and pool = 'BTC.BTC'
limit 50
