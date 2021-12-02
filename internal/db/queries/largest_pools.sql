-- Largest pools by rune depth for the latest blocks.
select
	pool,
	to_timestamp(block_timestamp / 1000000000)::date as date,
	rune_e8,
	block_timestamp
from block_pool_depths
order by block_timestamp desc, rune_e8 desc limit 1000