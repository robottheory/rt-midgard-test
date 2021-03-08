package record

// This file contains temporary hacks for when thornode is lacking events or sending extra events.

const poolCycle = 1000
const stagedPoolCost = 1000000000

// TODO(acsaba): remove when these events are emitted.
// https://gitlab.com/thorchain/thornode/-/issues/829
//
// In current testnet stagedPoolCost evetns are not emmitted. This can be removed once that happens.
// https://testnet.thornode.thorchain.info/thorchain/constants
// Every PoolCycle(1000) rune is deducted: StagedPoolCost(1000000000) .
// This emulates it.
func FixStagedPoolCost(height int64) {
	if height%poolCycle == 0 {
		for _, stagedPool := range stagedPools {
			if stagedPool.hasRuneFirst <= height && height <= stagedPool.availableHeight {
				Recorder.AddPoolRuneE8Depth([]byte(stagedPool.Pool), -stagedPoolCost)
			}

		}
	}
}

type poolStagedInterval struct {
	Pool            string
	hasRuneFirst    int64
	availableHeight int64
}

// This table can be generated with the following sql query:
//
// select FORMAT('	   {"%s", %s, %s},', first_available.pool, first_depth_height, available_height)
// from
//     (select pool, first_rune, height as first_depth_height
//     from
//         (select
//             pool,
//             first(rune_e8, block_timestamp) as first_rune,
//             min(block_timestamp) as block_timestamp
//         from block_pool_depths group by pool) as a
//     join block_log on block_timestamp = timestamp) as first_depth
// join
//     (select asset as pool, height as available_height
//     from pool_events
//     join block_log on timestamp=block_timestamp
//     where status='Available') as first_available
// on first_available.pool = first_depth.pool;
var stagedPools = []poolStagedInterval{
	{"ETH.ETH", 377, 377},
	{"ETH.USDT-0X62E273709DA575835C7F6AEF4A31140CA5B1D190", 447, 3000},
	{"BNB.BNB", 448, 448},
	{"BNB.BUSD-BAF", 612, 2000},
	{"BNB.USDT-DC8", 646, 1000},
	{"LTC.LTC", 726, 726},
	{"BTC.BTC", 729, 729},
	{"BCH.BCH", 1073, 1073},
}
