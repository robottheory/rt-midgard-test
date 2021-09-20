select distinct on(pool, rune_addr)
	pool,
	coalesce(rune_addr, '') as rune_addr,
	coalesce(last_value(asset_addr) over wnd, '') as asset_addr,
	coalesce(last_value(added_asset_e8) over wnd, 0) as added_asset_e8,
	coalesce(last_value(added_rune_e8) over wnd, 0) as added_rune_e8,
	coalesce(last_value(withdrawn_asset_e8) over wnd, 0) as withdrawn_asset_e8,
    coalesce(last_value(withdrawn_rune_e8) over wnd, 0) as withdrawn_rune_e8,
	coalesce(last_value(added_stake) over wnd, 0) -
	    coalesce(last_value(withdrawn_stake) over wnd, 0) as liquidity_units,
	coalesce(min_add_timestamp / 1000000000, 0) as first_add_date,
	coalesce(max_add_timestamp / 1000000000, 0) as last_add_date
from (
	select
		pool,
		rune_addr,
		min(asset_addr) as asset_addr,
		asset_addr_partition,
		sum(added_asset_e8) as added_asset_e8,
        sum(added_rune_e8) as added_rune_e8,
		sum(added_stake) as added_stake,
		sum(withdrawn_asset_e8) as withdrawn_asset_e8,
		sum(withdrawn_rune_e8) as withdrawn_rune_e8,
		sum(withdrawn_stake) as withdrawn_stake,
		min(add_timestamp) as min_add_timestamp,
		max(add_timestamp) as max_add_timestamp
	from (
		select
			coalesce(stake.pool, unstake.pool) as pool,
			stake.block_timestamp as add_timestamp,
			coalesce(stake.rune_addr, unstake.from_addr) as rune_addr,
			asset_addr,
			stake.rune_e8 as added_rune_e8,
			stake.asset_e8 as added_asset_e8,
			stake.stake_units as added_stake,
			unstake.emit_rune_e8 as withdrawn_rune_e8,
			unstake.emit_asset_e8 as withdrawn_asset_e8,
			unstake.stake_units as withdrawn_stake,
			coalesce(
				sum(case when unstake.basis_points = 10000 then 1 else 0 end)
				over (partition by coalesce(stake.pool, unstake.pool),
					               coalesce(stake.rune_addr, unstake.from_addr)
					order by coalesce(stake.block_timestamp, unstake.block_timestamp)
					rows between unbounded preceding and 1 preceding), 0) as asset_addr_partition
		from midgard.stake_events as stake full outer join
		   (select * from midgard.unstake_events) as unstake
		on stake.block_timestamp = unstake.block_timestamp
		order by pool, coalesce(stake.block_timestamp, unstake.block_timestamp) asc) as timeseries
	group by pool, rune_addr, asset_addr_partition
	order by pool, asset_addr_partition) as rolled_up
window wnd as (partition by pool, rune_addr order by asset_addr_partition
				rows between unbounded preceding and unbounded following)
limit 100