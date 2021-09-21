select
	*,
	m2m_current_stake_in_asset_e8
		+ m2m_withdrawn_rune_in_asset_e8
		+ withdrawn_asset_e8 as m2m_final_position_asset_e8
from (
	select
		*,
		stake_fraction * rune_depth_e8 as staked_rune_e8,
		stake_fraction * asset_depth_e8 as staked_asset_e8,
		stake_fraction * rune_depth_e8 - added_rune_e8 as return_rune_e8,
		stake_fraction * asset_depth_e8 - added_asset_e8 as return_asset_e8,
		stake_fraction * asset_depth_e8 +
			(stake_fraction * rune_depth_e8) / last_asset_to_rune_e8
			as m2m_current_stake_in_asset_e8
	from (
		select
			*,
			sum(stake) over (partition by pool) as total_pool_stake,
			stake / sum(stake) over (partition by pool) as stake_fraction
		from member_details
		where pool in ('BNB.BNB', 'BTC.BTC', 'ETH.ETH')
		order by substring(member_addr, 10, 10) desc
		limit 300
	) as md
	left outer join last_pool_depths using (pool)
	left outer join (
		select
			last_asset_to_rune_e8 as last_usd_to_rune_e8
		from last_pool_depths where pool='BNB.BUSD-BD1'
	) as usd on 1=1
) as rets
