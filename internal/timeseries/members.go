package timeseries

import (
	"context"
	"fmt"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// Represents membership in a pool
type membership struct {
	runeAddress    string
	assetAddress   string
	liquidityUnits int64
}

type addrIndex map[string](map[string]*membership)

func (index addrIndex) getMembership(address, pool string) (*membership, bool) {
	_, ok := index[address]
	if ok {
		ret, ok := index[address][pool]
		return ret, ok
	} else {
		return nil, false
	}
}

func (index addrIndex) setMembership(address, pool string, newMembership *membership) {
	if index[address] == nil {
		index[address] = make(map[string]*membership)
	}
	index[address][pool] = newMembership
}

// MemberAddrs gets all member known addresses.
// When there's a rune/asset address pair or a rune addres for the member,
// the rune asset is shown.
// Else the asset address is shown.
// If an address participates in multiple pools it will be shown only once
func GetMemberAddrs(ctx context.Context, pool *string) (addrs []string, err error) {
	// Build indexes: nested maps -> address and pools for each address as keys
	// Needed to access each member from any address and also to identify unique addresses

	// runeAddrIndex: all memberships with a rune address
	// using the rune address as key
	runeAddrIndex := make(addrIndex)

	// asymAddrIndex: all memberships with only an asset address
	// none of the pointes here should be stored in runeAddrIndex
	// A single asset address can stake in different pools
	// (E.g.: ETH address in mutiple ERC20 tokens)
	asymAssetAddrIndex := make(addrIndex)

	poolFilter := ""
	qargs := []interface{}{}
	if pool != nil {
		poolFilter = "pool = $1"
		qargs = append(qargs, pool)
	}

	// Rune asset queryies. If a liquidity provider has a rune address then it is identified
	// by its rune address.
	// NOTE: Assumes only a single asset address per pool can be paired with a single rune
	// address
	runeALQ := `
		SELECT
			rune_addr,
			COALESCE(MAX(asset_addr), ''),
			pool,
			SUM(stake_units) as liquidity_units
		FROM stake_events
		` + db.Where(poolFilter, "rune_addr IS NOT NULL") + `
		GROUP BY rune_addr, pool
	`
	runeALRows, err := db.Query(ctx, runeALQ, qargs...)
	if err != nil {
		return nil, err
	}
	defer runeALRows.Close()

	for runeALRows.Next() {
		var newMembership membership
		var pool string
		err := runeALRows.Scan(
			&newMembership.runeAddress,
			&newMembership.assetAddress,
			&pool,
			&newMembership.liquidityUnits)
		if err != nil {
			return nil, err
		}
		runeAddrIndex.setMembership(newMembership.runeAddress, pool, &newMembership)
	}

	// Asymmetrical addLiquidity with asset only
	// part of asym membership (as if there was a rune address present, the liquidity provider
	// would be matched using the rune address)
	asymAssetALQ := `
		SELECT
			asset_addr,
			pool,
			SUM(stake_units) as liquidity_units
		FROM stake_events
		` + db.Where(poolFilter, "asset_addr IS NOT NULL AND rune_addr IS NULL") + `
		GROUP BY asset_addr, pool
	`

	asymAssetALRows, err := db.Query(ctx, asymAssetALQ, qargs...)
	if err != nil {
		return nil, err
	}
	defer asymAssetALRows.Close()
	for asymAssetALRows.Next() {
		var assetAddress, pool string
		var liquidityUnits int64
		err := asymAssetALRows.Scan(&assetAddress, &pool, &liquidityUnits)
		if err != nil {
			return nil, err
		}
		newMembership := membership{
			assetAddress:   assetAddress,
			liquidityUnits: liquidityUnits,
		}
		asymAssetAddrIndex.setMembership(assetAddress, pool, &newMembership)
	}

	// Withdraws: try matching from address to a membreship from
	// the index and subtract addLiquidityUnits.
	// If there's no match either there's an error with the
	// implementation or the Thorchain events.
	withdrawQ := `
		SELECT
			from_addr,
			pool,
			SUM(stake_units) as liquidity_units
		FROM unstake_events
		` + db.Where(poolFilter) + `
		GROUP BY from_addr, pool
	`
	withdrawRows, err := db.Query(ctx, withdrawQ, qargs...)
	if err != nil {
		return nil, err
	}
	defer withdrawRows.Close()

	for withdrawRows.Next() {
		var fromAddr, pool string
		var liquidityUnits int64
		err := withdrawRows.Scan(&fromAddr, &pool, &liquidityUnits)
		if err != nil {
			return nil, err
		}

		existingMembership, ok := runeAddrIndex.getMembership(fromAddr, pool)
		if ok && (existingMembership.runeAddress == fromAddr) {
			existingMembership.liquidityUnits -= liquidityUnits
			continue
		}

		existingMembership, ok = asymAssetAddrIndex.getMembership(fromAddr, pool)
		if ok && (existingMembership.assetAddress == fromAddr) {
			existingMembership.liquidityUnits -= liquidityUnits
			continue
		}

		return nil, fmt.Errorf("Address %s, pool %s, found in withdraw events should have a matching membership", fromAddr, pool)
	}

	// Lookup membership addresses:
	// Either in runeIndex or asymIndex with at least one pool
	// with positive liquidityUnits balance
	addrs = make([]string, 0, len(runeAddrIndex)+len(asymAssetAddrIndex))

	for address, poolMemberships := range runeAddrIndex {
		// if it has at least a non zero balance, add it to the result
		isMember := false
		for _, memb := range poolMemberships {
			if memb.liquidityUnits > 0 {
				isMember = true
				break
			}
		}

		if isMember {
			addrs = append(addrs, address)
		}
	}

	for address, poolMemberships := range asymAssetAddrIndex {
		// if it has at least a non zero balance, add it to the result
		isMember := false
		for _, memb := range poolMemberships {
			if memb.liquidityUnits > 0 {
				isMember = true
				break
			}
		}

		if isMember {
			addrs = append(addrs, address)
		}
	}

	return addrs, nil
}

// Info of a member in a specific pool.
type MemberPool struct {
	Pool           string
	RuneAddress    string
	AssetAddress   string
	LiquidityUnits int64
	RuneAdded      int64
	AssetAdded     int64
	RunePending    int64
	AssetPending   int64
	DateFirstAdded int64
	DateLastAdded  int64
	RuneWithdrawn  int64
	AssetWithdrawn int64
}

func (memberPool MemberPool) toOapigen() oapigen.MemberPool {
	return oapigen.MemberPool{
		Pool:           memberPool.Pool,
		RuneAddress:    memberPool.RuneAddress,
		AssetAddress:   memberPool.AssetAddress,
		LiquidityUnits: util.IntStr(memberPool.LiquidityUnits),
		RuneAdded:      util.IntStr(memberPool.RuneAdded),
		AssetAdded:     util.IntStr(memberPool.AssetAdded),
		RuneWithdrawn:  util.IntStr(memberPool.RuneWithdrawn),
		AssetWithdrawn: util.IntStr(memberPool.AssetWithdrawn),
		RunePending:    util.IntStr(memberPool.RunePending),
		AssetPending:   util.IntStr(memberPool.AssetPending),
		DateFirstAdded: util.IntStr(memberPool.DateFirstAdded),
		DateLastAdded:  util.IntStr(memberPool.DateLastAdded),
	}
}

// Pools data associated with a single member
type MemberPools []MemberPool

func (memberPools MemberPools) ToOapigen() []oapigen.MemberPool {
	ret := make([]oapigen.MemberPool, len(memberPools))
	for i, memberPool := range memberPools {
		ret[i] = memberPool.toOapigen()
	}

	return ret
}

func GetMemberPools(ctx context.Context, address string) (MemberPools, error) {
	if record.AddressIsRune(address) {
		return memberDetailsRune(ctx, address)
	} else {
		return memberDetailsAsset(ctx, address)
	}
}

const mpPendingQFields = `
		COALESCE(SUM(asset_e8), 0),
		COALESCE(SUM(rune_e8), 0)
`

func getMemberDetailsQuery(address string) string {
	var address_where_clause string
	if record.AddressIsRune(address) {
		address_where_clause = ` where member_addr = $1 `
	} else {
		address_where_clause = ` where asset_addr = $1 `
	}
	// Aggregate the add- and withdraw- liquidity events. Conceptually we need to
	// union the stake_events and unstake_events tables and aggregate the add
	// and withdrawal amounts grouping by pool and member id. In practice the
	// query gets a bit complicated because it needs to account for situations
	// like the following:
	//   1. liquidity is added symmetrically
	//   2. all of the asset is withdrawn
	// In this case, the asset address should be forgotten. To achieve this,
	// the events are assigned a partition number which is incremented each time
	// all assets are withdrawn (i.e. basis_points=10000 for the withdrawal event).
	// Then, when the events are aggregated, they are grouped by pool, rune_address
	// and partition number, with only the rows with the highest partition number
	// for each pool/rune_address group returned.
	return `
		-- View of the union of stake and unstake events.
		with stake_unstake_events as (
			select
				pool,
				block_timestamp,
				rune_addr,
				asset_addr,
				coalesce(rune_addr, asset_addr) as member_addr,
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
					cast(NULL as text) as rune_addr,
					cast(NULL as text) as asset_addr,
					from_addr as member_addr,
					cast(NULL as BigInt) as added_rune_e8,
					cast(NULL as BigInt) as added_asset_e8,
					cast(NULL as BigInt) as added_stake,
					emit_rune_e8 as withdrawn_rune_e8,
					emit_asset_e8 as withdrawn_asset_e8,
					stake_units as withdrawn_stake,
					basis_points as withdrawn_basis_points
				from midgard.unstake_events
			)
		),
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
		events_with_partition as (
			select *,
				coalesce(
					sum(case when withdrawn_basis_points = 10000 then 1 else 0 end)
					over (partition by pool, member_addr
						order by block_timestamp
						rows between unbounded preceding and 1 preceding), 0) as asset_addr_partition
			from stake_unstake_events
		),
		-- Aggregate added and withdrawn liquidity for each member.
		aggregated_members as (
			select
				pool,
				member_addr,
				min(rune_addr) as rune_addr,
				min(asset_addr) as asset_addr,
				asset_addr_partition,
				sum(added_asset_e8) as added_asset_e8,
				sum(added_rune_e8) as added_rune_e8,
				sum(withdrawn_asset_e8) as withdrawn_asset_e8,
				sum(withdrawn_rune_e8) as withdrawn_rune_e8,
				sum(added_stake) as added_stake,
				sum(withdrawn_stake) as withdrawn_stake,
				min(block_timestamp) filter (where added_stake > 0) as min_add_timestamp,
				max(block_timestamp) filter (where added_stake > 0) as max_add_timestamp
			from events_with_partition
			group by pool, member_addr, asset_addr_partition
			order by pool, asset_addr_partition
		)
		-- Select the last member for each rune address.
		select distinct on(pool, member_addr)
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
		from aggregated_members
		` + address_where_clause + `
		window wnd as (partition by pool, member_addr order by asset_addr_partition
						rows between unbounded preceding and unbounded following)
		order by pool, member_addr`
}

// RUNE addresses
func memberDetailsRune(ctx context.Context, runeAddress string) (MemberPools, error) {
	memberDetailsQ := getMemberDetailsQuery(runeAddress)

	rows, err := db.Query(ctx, memberDetailsQ, runeAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memberPoolsMap := make(map[string]MemberPool)

	for rows.Next() {
		memberPool := MemberPool{}
		err := rows.Scan(
			&memberPool.Pool,
			&memberPool.RuneAddress,
			&memberPool.AssetAddress,
			&memberPool.AssetAdded,
			&memberPool.RuneAdded,
			&memberPool.AssetWithdrawn,
			&memberPool.RuneWithdrawn,
			&memberPool.LiquidityUnits,
			&memberPool.DateFirstAdded,
			&memberPool.DateLastAdded,
		)
		if err != nil {
			return nil, err
		}
		memberPoolsMap[memberPool.Pool] = memberPool
	}

	pendingLiquidityQ := `SELECT
		pool,
	` + mpPendingQFields + `
	FROM midgard_agg.pending_adds
	WHERE rune_addr = $1
	GROUP BY pool`

	pendingLiquidityRows, err := db.Query(ctx, pendingLiquidityQ, runeAddress)
	if err != nil {
		return nil, err
	}
	defer pendingLiquidityRows.Close()

	for pendingLiquidityRows.Next() {
		var pool string
		var assetE8, runeE8 int64

		err := pendingLiquidityRows.Scan(
			&pool,
			&assetE8,
			&runeE8,
		)
		if err != nil {
			return nil, err
		}

		memberPool, ok := memberPoolsMap[pool]
		if !ok {
			memberPool.Pool = pool
			memberPool.RuneAddress = runeAddress
		}

		memberPool.AssetPending = assetE8
		memberPool.RunePending = runeE8
		memberPoolsMap[memberPool.Pool] = memberPool
	}

	ret := make(MemberPools, 0, len(memberPoolsMap))
	for _, memberPool := range memberPoolsMap {
		if memberPool.LiquidityUnits > 0 ||
			0 < memberPool.AssetPending || 0 < memberPool.RunePending {
			ret = append(ret, memberPool)
		}
	}

	return ret, nil
}

func memberDetailsAsset(ctx context.Context, assetAddress string) (MemberPools, error) {
	memberDetailsQ := getMemberDetailsQuery(`WHERE asset_addr = $1`)

	rows, err := db.Query(ctx, memberDetailsQ, assetAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memberPools MemberPools
	for rows.Next() {
		memberPool := MemberPool{}
		err := rows.Scan(
			&memberPool.Pool,
			&memberPool.RuneAddress,
			&memberPool.AssetAddress,
			&memberPool.AssetAdded,
			&memberPool.RuneAdded,
			&memberPool.AssetWithdrawn,
			&memberPool.RuneWithdrawn,
			&memberPool.LiquidityUnits,
			&memberPool.DateFirstAdded,
			&memberPool.DateLastAdded,
		)
		if err != nil {
			return nil, err
		}

		var whereAddLiquidityAddresses, queryAddress string
		if memberPool.RuneAddress == "" {
			// asym liquidity provider, asset address is used to identify it
			// (if there is a rune_addr it will always be used to get the lp so it has to be NULL)
			whereAddLiquidityAddresses = "WHERE asset_addr = $1 AND rune_addr IS NULL"
			queryAddress = memberPool.AssetAddress
		} else {
			// sym liquidity provider, rune address is used to identify it
			whereAddLiquidityAddresses = "WHERE rune_addr = $1"
			queryAddress = memberPool.RuneAddress
		}

		pendingLiquidityQ := `SELECT ` + mpPendingQFields + `FROM midgard_agg.pending_adds ` + whereAddLiquidityAddresses + ` AND pool = $2`

		pendingLiquidityRow, err := db.Query(ctx, pendingLiquidityQ, queryAddress, memberPool.Pool)
		if err != nil {
			return nil, err
		}
		defer pendingLiquidityRow.Close()
		if pendingLiquidityRow.Next() {
			err := pendingLiquidityRow.Scan(&memberPool.AssetPending, &memberPool.RunePending)
			if err != nil {
				return nil, err
			}
		}

		if memberPool.LiquidityUnits > 0 {
			memberPools = append(memberPools, memberPool)
		}
	}

	return memberPools, nil
}
