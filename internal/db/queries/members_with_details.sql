-- View of the union of stake and unstake events.
with stake_unstake_events as (select
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
)),

-- View of the union of stake and unstake events with an additional column to disambiguate
-- members having the same rune address.
--
-- A single rune address may be used to provide liquidity to a single pool multiple times.
-- For example, if liquidity is added symmetrically, then completely removed, liquidity
-- can be added again with the same rune address, but a different asset address.
-- After liquidity is completely removed, the associated asset address should be forgotten.
-- Further, the added (and removed) liquidity should also not be associated with the rune
-- address if it is used again. To accomplish this, we order the stake and unstake events
-- by block timestamp, and each time the liquidity for a given pool and a given rune
-- address is completely withdrawn (basis_points=10000), we increment a counter. This
-- counter partitions the events so that they can be grouped together and viewed as
-- corresponding to distinct members.
--
-- Implementation detail: the partition count is computed as a cumulative sum over
-- and indicator function: basis_points=10000 ? 1 : 0. The cumulative sum is computed
-- over all rows up to, but not including the current row. The current row is excluded
-- to ensure that the event corresponding to the withdrawl is grouped together with
-- the other events corresponding to that member (and not the next one).
events_with_partition as (select *,
	coalesce(
		sum(case when withdrawn_basis_points = 10000 then 1 else 0 end)
		over (partition by pool, member_addr
			order by block_timestamp
			  rows between unbounded preceding and 1 preceding), 0) as asset_addr_partition
from stake_unstake_events),

-- Liquidity events together with the pool depths and asset prices at the respective
-- block timestamps.
events_with_partition_and_pool_depth_and_prices as (select
	events_with_partition.*,
	cast(pd.rune_e8 as decimal) / pd.asset_e8 as asset_to_rune_e8,
	cast(usd.rune_e8 as decimal) / usd.asset_e8 as usd_to_rune_e8
from events_with_partition
left outer join midgard.block_pool_depths as pd using (block_timestamp, pool)
left outer join (
	select * from midgard.block_pool_depths
	where pool='BNB.BUSD-BD1'
) as usd using (block_timestamp)
where pd.asset_e8 != 0 and usd.asset_e8 != 0),

-- Aggregate added and withdrawn liquidity for each member with mark-to-market valuations.
aggregated_members as (select
		pool,
		member_addr,
		min(asset_addr) as asset_addr,
		asset_addr_partition,
		sum(added_asset_e8) as added_asset_e8,
		sum(added_rune_e8) as added_rune_e8,
		sum(added_stake) as added_stake,
		sum(added_asset_e8 * asset_to_rune_e8) as m2m_added_asset_in_rune_e8,
		sum(added_rune_e8 / asset_to_rune_e8) as m2m_added_rune_in_asset_e8,
		sum(added_asset_e8 * asset_to_rune_e8 / usd_to_rune_e8) as m2m_added_asset_in_usd_e8,
		sum(added_rune_e8 / usd_to_rune_e8) as m2m_added_rune_in_usd_e8,
		sum(withdrawn_asset_e8) as withdrawn_asset_e8,
		sum(withdrawn_rune_e8) as withdrawn_rune_e8,
		sum(withdrawn_stake) as withdrawn_stake,
		sum(withdrawn_asset_e8 * asset_to_rune_e8) as m2m_withdrawn_asset_in_rune_e8,
		sum(withdrawn_rune_e8 / asset_to_rune_e8) as m2m_withdrawn_rune_in_asset_e8,
		sum(withdrawn_asset_e8 * asset_to_rune_e8 / usd_to_rune_e8) as m2m_withdrawn_asset_in_usd_e8,
		sum(withdrawn_rune_e8 / usd_to_rune_e8) as m2m_withdrawn_rune_in_usd_e8,
		min(block_timestamp) filter (where added_stake > 0) as min_add_timestamp,
		max(block_timestamp) filter (where added_stake > 0) as max_add_timestamp
from events_with_partition_and_pool_depth_and_prices
where asset_to_rune_e8 != 0 and usd_to_rune_e8 != 0
group by pool, member_addr, asset_addr_partition
order by pool, asset_addr_partition),

-- Select the last member for each rune address.
member_details as (select distinct on(pool, member_addr)
	pool,
	coalesce(member_addr, '') as member_addr,
	coalesce(last_value(asset_addr) over wnd, '') as asset_addr,
	coalesce(last_value(added_asset_e8) over wnd, 0) as added_asset_e8,
	coalesce(last_value(added_rune_e8) over wnd, 0) as added_rune_e8,
	coalesce(last_value(withdrawn_asset_e8) over wnd, 0) as withdrawn_asset_e8,
	coalesce(last_value(withdrawn_rune_e8) over wnd, 0) as withdrawn_rune_e8,
	coalesce(last_value(m2m_added_asset_in_rune_e8) over wnd, 0) as m2m_added_asset_in_rune_e8,
	coalesce(last_value(m2m_added_rune_in_asset_e8) over wnd, 0) as m2m_added_rune_in_asset_e8,
	coalesce(last_value(m2m_added_asset_in_usd_e8) over wnd, 0) as m2m_added_asset_in_usd_e8,
	coalesce(last_value(m2m_added_rune_in_usd_e8) over wnd, 0) as m2m_added_rune_in_usd_e8,	
	coalesce(last_value(m2m_withdrawn_asset_in_rune_e8) over wnd, 0) as m2m_withdrawn_asset_in_rune_e8,
	coalesce(last_value(m2m_withdrawn_rune_in_asset_e8) over wnd, 0) as m2m_withdrawn_rune_in_asset_e8,
	coalesce(last_value(m2m_withdrawn_asset_in_usd_e8) over wnd, 0) as m2m_withdrawn_asset_in_usd_e8,
	coalesce(last_value(m2m_withdrawn_rune_in_usd_e8) over wnd, 0) as m2m_withdrawn_rune_in_usd_e8,
	coalesce(last_value(added_stake) over wnd, 0) -
		coalesce(last_value(withdrawn_stake) over wnd, 0) as stake,
	to_timestamp(min_add_timestamp / 1000000000)::date as first_add_date,
	to_timestamp(max_add_timestamp / 1000000000)::date as last_add_date
from aggregated_members
window wnd as (partition by pool, member_addr order by asset_addr_partition
				rows between unbounded preceding and unbounded following)),

-- Select the pool depths at the latest block timestamp available for each pool.
last_pool_depths as (select
	pool,
	rune_depth_e8,
	asset_depth_e8,
	last_asset_to_rune_e8,
	latest_pool_depth_block_timestamp
from (select
		pool,
		rune_e8 as rune_depth_e8,
		asset_e8 as asset_depth_e8,
		cast(rune_e8 as decimal) / asset_e8 as last_asset_to_rune_e8,
	  	block_timestamp as latest_pool_depth_block_timestamp,
		row_number() over (partition by pool
						   order by block_timestamp desc) as r
	from block_pool_depths
	where asset_e8 != 0) as sequenced
where r = 1),

last_total_stake as (select
		pool,
		latest_pool_stake_block_timestamp,
		total_stake
	from (select
			pool,
			block_timestamp as latest_pool_stake_block_timestamp,
			sum(coalesce(added_stake, 0) - coalesce(withdrawn_stake, 0)) over (partition by pool order by block_timestamp) as total_stake,
			row_number() over (partition by pool
							   order by block_timestamp desc) as r
		  from stake_unstake_events) as sequenced
where r = 1),

member_details_with_latest_pool_info as (select
	* from member_details
	full outer join last_total_stake using (pool)
	full outer join last_pool_depths using (pool)
	full outer join (
		select
			rune_e8::decimal / asset_e8 as usd_to_rune_e8,
			block_timestamp as usd_to_rune_block_timestamp
		from block_pool_depths
		where pool='BNB.BUSD-BD1') as pd
		on usd_to_rune_block_timestamp = latest_pool_depth_block_timestamp
),

metrics as (select
	*,
	stake::decimal / total_stake * asset_depth_e8 as redeamable_asset_e8,
	stake::decimal / total_stake * rune_depth_e8 as redeamable_rune_e8
from member_details_with_latest_pool_info
where total_stake != 0),

metrics2 as (select
	*,
	--redeamable_asset_e8::decimal / asset_depth_e8 * rune_depth_e8 as redeamable_asset_in_rune_e8,
	redeamable_asset_e8::decimal / asset_depth_e8 * rune_depth_e8 / usd_to_rune_e8 as redeamable_asset_in_usd_e8,
	redeamable_rune_e8::decimal / usd_to_rune_e8 as redeamable_rune_in_usd_e8
	from metrics
),

metrics3 as (select
	*,
	redeamable_asset_e8 + withdrawn_asset_e8 - added_asset_e8 as return_asset_e8,
	redeamable_rune_e8 + withdrawn_rune_e8 - added_rune_e8 as return_rune_e8,
	redeamable_asset_in_usd_e8 + m2m_withdrawn_asset_in_usd_e8 - m2m_added_asset_in_usd_e8 as return_asset_in_usd_e8,
	redeamable_rune_in_usd_e8 + m2m_withdrawn_rune_in_usd_e8 - m2m_added_rune_in_usd_e8 as return_rune_in_usd_e8
	from metrics2
),

metrics4 as (select
	*,
	return_asset_in_usd_e8 + return_rune_in_usd_e8 as return_usd_e8,
	(return_asset_in_usd_e8 + return_rune_in_usd_e8) / (m2m_added_asset_in_usd_e8::bigint + m2m_added_rune_in_usd_e8) as return_usd_pct
	from metrics3
)

select * from metrics4
order by stake desc
limit 1000