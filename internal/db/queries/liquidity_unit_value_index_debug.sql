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
			emit_asset_e8
				+ case when asset = pool and block_timestamp < 1630938167525799522
					then -asset_e8 else 0 end as withdrawn_asset_e8,
			stake_units as withdrawn_stake,
			imp_loss_protection_e8,
			basis_points as withdrawn_basis_points
		from midgard.unstake_events
		where pool = 'BTC.BTC'
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
--	* aggregate reward amounts
--	* aggregate gas amounts
--	* aggregate fee amounts
--	* aggregate swap amounts
--  * aggregate pool balance change amounts
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
		where pool = 'BTC.BTC'
	) as depths
	using (pool, block_timestamp)
    full outer join (
		select
			pool,
			block_timestamp,
			sum(rune_e8) as reward_rune_e8
		from midgard.rewards_event_entries
		where pool = 'BTC.BTC'
        group by pool, block_timestamp
	) as reward_amounts
	using (pool, block_timestamp)
	full outer join (
		select
			asset as pool,
			sum(asset_e8) as gas_event_asset_e8,
			sum(rune_e8) as gas_event_rune_e8,
			block_timestamp
		from midgard.gas_events
		where asset = 'BTC.BTC'
		group by asset, block_timestamp
	) as gas_amounts
	using (pool, block_timestamp)
	full outer join (
		select
			asset as pool,
			sum(asset_e8) as fee_event_asset_e8,
			sum(pool_deduct) as fee_event_rune_e8,
			block_timestamp
		from midgard.fee_events
		where asset = 'BTC.BTC'
		group by asset, block_timestamp
	) as fee_amounts
	using (pool, block_timestamp)
	full outer join (
		select
			pool,
			block_timestamp,
			-- _direction: 0=RuneToAsset 1=AssetToRune 2=RuneToSynth 3=SynthToRune
			-- So, for _direction=0, the RUNE pool depth increases by from_e8.
			sum(case when _direction = 0 then from_e8 else 0 end)
				+ sum(case when _direction = 1 then -to_e8 else 0 end) as swap_added_rune_e8,
			sum(case when _direction = 1 then from_e8 else 0 end)
				+ sum(case when _direction = 0 then -to_e8 else 0 end) as swap_added_asset_e8
		from swap_events
		group by pool, block_timestamp
	) as swap_amounts
	using (pool, block_timestamp)
    full outer join (
		select
			pool,
			block_timestamp,
			sum(case when asset = 'THOR.RUNE' then asset_e8 else 0 end) as slash_rune_e8,
			sum(case when asset != 'THOR.RUNE' then asset_e8 else 0 end) as slash_asset_e8
		from slash_amounts
		group by pool, block_timestamp
	) as slash_amounts
	using (pool, block_timestamp)
	full outer join (
		select
			asset as pool,
			block_timestamp,
			sum(case when rune_add = true then rune_amt else -rune_amt end) as pool_balance_chg_rune_add,
			sum(case when asset_add = true then asset_amt else -asset_amt end) as pool_balance_chg_asset_add
		from pool_balance_change_events
		group by pool, block_timestamp
	) as pool_balance_change_amounts
	using (pool, block_timestamp)
),
-- Summary of events per block together with total stake.
blocks as (
	select
		pool,
		block_timestamp,
		to_timestamp(block_timestamp / 1000000000)::date as date,
		depth_asset_e8,
		depth_rune_e8,
		coalesce(added_rune_e8, 0) as added_rune_e8,
        coalesce(added_asset_e8, 0) as added_asset_e8,
		coalesce(added_stake, 0) as added_stake,
		coalesce(withdrawn_rune_e8, 0) as withdrawn_rune_e8,
		coalesce(withdrawn_asset_e8, 0) as withdrawn_asset_e8,
		coalesce(withdrawn_stake, 0) as withdrawn_stake,
		coalesce(imp_loss_protection_e8, 0) as imp_loss_protection_e8,
		coalesce(swap_added_asset_e8, 0) as swap_added_asset_e8,
		coalesce(swap_added_rune_e8, 0) as swap_added_rune_e8,
		coalesce(reward_rune_e8, 0) as reward_rune_e8,
		coalesce(gas_event_asset_e8, 0) as gas_event_asset_e8,
		coalesce(gas_event_rune_e8, 0) as gas_event_rune_e8,
		coalesce(fee_event_asset_e8, 0) as fee_event_asset_e8,
		coalesce(fee_event_rune_e8, 0) as fee_event_rune_e8,
		coalesce(slash_rune_e8, 0) as slash_rune_e8,
		coalesce(slash_asset_e8, 0) as slash_asset_e8,
		coalesce(pool_balance_chg_rune_add, 0) as pool_balance_chg_rune_add,
		coalesce(pool_balance_chg_asset_add, 0) as pool_balance_chg_asset_add,
		sum(coalesce(added_stake, 0)) over cumulative_wnd
			- sum(coalesce(withdrawn_stake, 0)) over cumulative_wnd as total_stake
	from block_summary
	window cumulative_wnd as (partition by pool order by block_timestamp)
),
-- Summary of events per block together with total stake and a check that the
-- rune and asset depths are what they are expected to be given the events.
blocks_with_check as (
	select
		*,
		lag(depth_asset_e8, 1) over wnd as prev_asset_depth_e8,
		lag(depth_rune_e8, 1) over wnd as prev_rune_depth_e8,
		-- The value below should always equal 0.
		depth_asset_e8 - lag(depth_asset_e8, 1) over wnd
			+ withdrawn_asset_e8 - added_asset_e8 - swap_added_asset_e8
			+ gas_event_asset_e8 - fee_event_asset_e8
			- slash_asset_e8 - pool_balance_chg_asset_add as asset_chg_check,
		-- The value below should always equal 0.
		depth_rune_e8 - lag(depth_rune_e8, 1) over wnd
			+ withdrawn_rune_e8 - added_rune_e8 - swap_added_rune_e8 - imp_loss_protection_e8
			+ fee_event_rune_e8 - gas_event_rune_e8 - reward_rune_e8
			- slash_rune_e8 - pool_balance_chg_rune_add as rune_chg_check
	from blocks
	window wnd as (partition by pool order by block_timestamp)
),
metrics as (
	select
		*,
		depth_asset_e8::numeric * depth_rune_e8 as depth_product,
		(depth_asset_e8::numeric + 1) *
			(depth_rune_e8
			- gas_event_rune_e8 +
			 	(gas_event_asset_e8::numeric * depth_rune_e8
				 / (depth_asset_e8 - gas_event_asset_e8))
			+ fee_event_rune_e8 - (fee_event_asset_e8::numeric * depth_rune_e8
				/ (depth_asset_e8 + fee_event_asset_e8)) + 1) as adjusted_depth_product
	from blocks_with_check
	order by block_timestamp
)
select * from (
	select
		pool,
		block_timestamp,
		to_timestamp(block_timestamp / 1000000000)::date as date,
	    asset_chg_check,
		rune_chg_check,
		added_rune_e8,
		added_asset_e8,
		added_stake,
		withdrawn_rune_e8,
		withdrawn_asset_e8,
		withdrawn_stake,
		imp_loss_protection_e8,
		reward_rune_e8,
		gas_event_asset_e8,
		gas_event_rune_e8,
		fee_event_asset_e8,
		fee_event_rune_e8,
		pool_balance_chg_asset_add,
		pool_balance_chg_rune_add,
		total_stake,
		depth_asset_e8,
		depth_rune_e8,
		lag(total_stake, 1) over wnd as prev_total_stake,
		lag(depth_asset_e8, 1) over wnd as prev_depth_asset_e8,
		lag(depth_rune_e8, 1) over wnd as prev_depth_rune_e8,
		sqrt(depth_product) / total_stake as liquidity_unit_value_index,
		lag(sqrt(depth_product) / total_stake, 1) over wnd as prev_liquidity_unit_value_index,
		sqrt(adjusted_depth_product) / total_stake < lag(sqrt(depth_product) / total_stake, 1) over wnd as luvi_decrease,
		sqrt(adjusted_depth_product) / total_stake
			/ lag(sqrt(depth_product) / total_stake, 1) over wnd - 1 as pct_change
	from metrics
	where pool = 'BTC.BTC'
	window wnd as (partition by pool order by block_timestamp)
) as summary
where luvi_decrease = true and pool = 'BTC.BTC'
limit 100
