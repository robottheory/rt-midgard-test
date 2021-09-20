select * from (
    select
		pool,
		week_timestamp,
		added_asset_e8,
		added_rune_e8,
		added_stake,
		withdrawn_asset_e8,
		withdrawn_rune_e8,
		withdrawn_stake,
		sum(coalesce(added_stake, 0) - coalesce(withdrawn_stake, 0)) over wnd as total_stake
	from (
		select
			pool,
			week_timestamp,
			sum(added_asset_e8) as added_asset_e8,
			sum(added_rune_e8) as added_rune_e8,
			sum(added_stake) as added_stake,
			sum(withdrawn_asset_e8) as withdrawn_asset_e8,
			sum(withdrawn_rune_e8) as withdrawn_rune_e8,
			sum(withdrawn_stake) as withdrawn_stake
		from (
			select
				coalesce(stake.pool, unstake.pool) as pool,
				coalesce(stake.block_timestamp,
						 unstake.block_timestamp) / 1000000000 / 7 as week_timestamp,
				stake.asset_e8 as added_asset_e8,
				stake.rune_e8 as added_rune_e8,
				stake.stake_units as added_stake,
				unstake.stake_units as withdrawn_stake,
				unstake.emit_asset_e8 as withdrawn_asset_e8,
				unstake.emit_rune_e8 as withdrawn_rune_e8
			from midgard.stake_events as stake full outer join
			   (select * from midgard.unstake_events
				where pool='ETH.ETH') as unstake
			on stake.block_timestamp = unstake.block_timestamp
			where stake.pool='ETH.ETH'
			order by pool, coalesce(stake.block_timestamp, unstake.block_timestamp) asc) as timeseries
		group by pool, week_timestamp
		order by week_timestamp) as rolled_up
	window wnd as (partition by pool order by week_timestamp asc)) as stake_units
left outer join (
	select pool,
	    last_value(asset_e8) over wnd as depth_asset_e8,
	    last_value(rune_e8) over wnd as depth_rune_e8,
	    last_value(block_timestamp / 1000000000 / 7) over wnd as week_timestamp
	from midgard.block_pool_depths
    where pool='ETH.ETH'
    window wnd as (partition by block_timestamp / 1000000000 / 7
				   order by block_timestamp)) as pool_depths
on stake_units.pool = pool_depths.pool
    and stake_units.week_timestamp = pool_depths.week_timestamp
