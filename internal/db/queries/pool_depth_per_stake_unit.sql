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
		order by block_timestamp
	)
),
-- View of stake and unstake events, together with pool depths.
stakes_and_depths as (
	select 
		*,
		sum(coalesce(added_stake, 0)) over wnd - sum(coalesce(withdrawn_stake, 0)) over wnd as total_stake,
		depth_asset_e8::numeric * depth_rune_e8 as depth_product
	from stake_unstake_events
	left outer join (
		select pool,
			asset_e8 as depth_asset_e8,
			rune_e8 as depth_rune_e8,
			block_timestamp
		from midgard.block_pool_depths
	) as depths
	using (pool, block_timestamp)
	where pool = 'BNB.BNB'
	window wnd as (partition by pool order by block_timestamp)
	order by block_timestamp
),
daily_stakes_and_depths as (
	select
		pool,
		block_timestamp,
		total_stake,
		depth_product,
		row_number() over (partition by pool, block_timestamp / 1000000000 / 60 / 60 / 24
						   order by block_timestamp desc) as r
	from stakes_and_depths
),
weekly_stakes_and_depths as (
	select
		pool,
		block_timestamp,
		total_stake,
		depth_product,
		row_number() over (partition by pool, block_timestamp / 1000000000 / 60 / 60 / 24 / 7
						   order by block_timestamp desc) as r
	from stakes_and_depths
)
select pool,
	to_timestamp(block_timestamp / 1000000000)::date as date,
	depth_product / total_stake / total_stake as depth_product_per_stake2,
	depth_product / total_stake / total_stake < lag(depth_product / total_stake / total_stake, 1)
		over (partition by pool order by block_timestamp) as decrease
from daily_stakes_and_depths
where r = 1
