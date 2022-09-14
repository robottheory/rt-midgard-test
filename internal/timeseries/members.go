package timeseries

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/thorchain/midgard/internal/db"
	"gitlab.com/thorchain/midgard/internal/fetch/record"
	"gitlab.com/thorchain/midgard/internal/util"
	"gitlab.com/thorchain/midgard/openapi/generated/oapigen"
)

// GetMemberIds returns the ids of all known members.
//
// The id of a member is defined as their rune address if they are participating with their rune
// address, or as their asset address otherwise (for members with asset address only.)
//
// Member ids present in multiple pools will be only returned once.
func GetMemberIds(ctx context.Context, pool *string) (addrs []string, err error) {
	poolFilter := ""

	qargs := []interface{}{}
	if pool != nil {
		poolFilter = "pool = $1"

		qargs = append(qargs, pool)
	}

	q := "SELECT DISTINCT member_id FROM midgard_agg.members " + db.Where(poolFilter)

	rows, err := db.Query(ctx, q, qargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var member string
		err := rows.Scan(&member)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, member)
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

// Info of a member in a specific pool.
type LPDetail struct {
	Pool              string
	LiquidityUnits    int64
	RuneAdded         int64
	AssetAdded        int64
	RuneWithdrawn     int64
	AssetWithdrawn    int64
	Date              int64
	Height            int64
	RunePriceUsd      float64
	AssetPriceUsd     float64
	AssetDepth        int64
	RuneDepth         int64
	PoolUnit          int64
	SharedUnit        int64
	AssetLiquidityFee int64
	RuneLiquidityFee  int64
	LiquidityFeeUsd   int64
	BlockRewards      int64
}

type SumUnit struct {
	Unit int64 `json:"unit"`
}

type PoolInfo struct {
	Pool      string  `json:"pool"`
	Date      int64   `json:"date"`
	PriceUsed float64 `json:"price_used"`
	AssetE8   int64   `json:"asset_e_8"`
	RuneE8    int64   `json:"rune_e_8"`
	Unit      int64   `json:"unit"`
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
	q := `
		SELECT
			pool,
			COALESCE(rune_addr, ''),
			COALESCE(asset_addr, ''),
			lp_units_total,
			added_rune_e8_total,
			added_asset_e8_total,
			withdrawn_rune_e8_total,
			withdrawn_asset_e8_total,
			pending_rune_e8_total,
			pending_asset_e8_total,
			COALESCE(first_added_timestamp / 1000000000, 0),
			COALESCE(last_added_timestamp / 1000000000, 0)
		FROM midgard_agg.members
		WHERE member_id = $1 OR asset_addr = $1
		ORDER BY pool`
	rows, err := db.Query(ctx, q, address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results MemberPools
	for rows.Next() {
		var entry MemberPool
		err := rows.Scan(
			&entry.Pool,
			&entry.RuneAddress,
			&entry.AssetAddress,
			&entry.LiquidityUnits,
			&entry.RuneAdded,
			&entry.AssetAdded,
			&entry.RuneWithdrawn,
			&entry.AssetWithdrawn,
			&entry.RunePending,
			&entry.AssetPending,
			&entry.DateFirstAdded,
			&entry.DateLastAdded,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, entry)
	}
	return results, nil
}

func GetFullMemberPools(ctx context.Context, address string) (MemberPools, error) {
	if record.AddressIsRune(address) {
		return memberDetailsRune(ctx, address)
	} else {
		memberPools, err := memberDetailsAsset(ctx, address)
		if err != nil {
			return memberPools, err
		}
		var runAddr string
		for _, memberPool := range memberPools {
			if memberPool.RuneAddress != "" {
				runAddr = memberPool.RuneAddress
			}
		}
		if runAddr != "" {
			runePools, err := GetMemberPools(ctx, runAddr)
			if err != nil {
				return memberPools, err
			}
			for _, p1 := range runePools {
				exits := false
				for _, p2 := range memberPools {
					if p1.Pool == p2.Pool && p1.RuneAddress == p2.RuneAddress && p1.AssetAddress == p2.AssetAddress {
						exits = true
						break
					}
				}
				if !exits {
					memberPools = append(memberPools, p1)
				}
			}
		}
		return memberPools, err
	}
}

func GetSumPoolUnits(ctx context.Context, pool string, date int64, tableName string) (*int64, *error) {
	q := `select sum(stake_units) from ` + tableName + ` where pool=$1 and block_timestamp<=$2`

	rows, err := db.Query(ctx, q, pool, date)
	if err != nil {
		return nil, &err
	}
	defer rows.Close()

	var sumUnit int64 = 0
	for rows.Next() {
		var entry SumUnit
		err := rows.Scan(
			&entry.Unit,
		)
		if err != nil {
			return nil, &err
		}
		sumUnit += entry.Unit
	}
	return &sumUnit, nil
}

func HotFixSumPoolUnit(ctx context.Context, pool string, lpDetails LPDetail) (*int64, *error) {

	var stack_event_unit int64 = 0
	var unStack_event_unit int64 = 0

	stakeUnit, err := GetSumPoolUnits(ctx, pool, lpDetails.Date, "stake_events")
	if err != nil {
		return nil, err
	}
	if stakeUnit != nil {
		stack_event_unit += *stakeUnit
	}

	unStakUnit, errr := GetSumPoolUnits(ctx, pool, lpDetails.Date, "unstake_events")
	if err != nil {
		return nil, errr
	}
	if unStakUnit != nil {
		unStack_event_unit += *unStakUnit
	}

	returnValue := stack_event_unit - unStack_event_unit
	return &returnValue, nil

}

func GetLpDetail(ctx context.Context, runeAddress, assetAddress, pool string) ([]LPDetail, error) {
	lpDetails, err := lpDetailsRune(ctx, runeAddress, assetAddress, pool)
	if err != nil {
		return nil, err
	}
	if len(lpDetails) == 0 {
		return lpDetails, nil
	}
	lpDetails, err = getBlockHeight(ctx, lpDetails)
	if err != nil {
		return nil, err
	}

	for i, items := range lpDetails {
		sumUnit, err := HotFixSumPoolUnit(ctx, pool, items)
		if err != nil {
			return nil, *err
		}
		lpDetails[i].PoolUnit = *sumUnit
	}

	for i, lp := range lpDetails {
		lpDetails[i].Date = lp.Date / 1000000000
	}

	lpDetails, err = poolInfo(ctx, pool, lpDetails)
	if err != nil {
		return nil, err
	}

	return lpDetails, nil
}

const mpAddLiquidityQFields = `
		COALESCE(SUM(asset_E8), 0),
		COALESCE(SUM(rune_E8), 0),
		COALESCE(SUM(stake_units), 0),
		COALESCE(MIN(block_timestamp) / 1000000000, 0),
		COALESCE(MAX(block_timestamp) / 1000000000, 0)
`

const lpAddLiquidityQFields = `
		asset_E8,
		rune_E8,
		stake_units,
		block_timestamp
`

const mpWithdrawQFields = `
		COALESCE(SUM(emit_asset_e8), 0),
		COALESCE(SUM(emit_rune_e8), 0),
		COALESCE(SUM(stake_units), 0)
`

const lpWithdrawQFields = `
		emit_asset_e8,
		emit_rune_e8,
		stake_units,
		block_timestamp
`

const mpPendingQFields = `
		COALESCE(SUM(asset_e8), 0),
		COALESCE(SUM(rune_e8), 0)
`

// RUNE addresses
func memberDetailsRune(ctx context.Context, runeAddress string) (MemberPools, error) {
	// If a member has a RUNE address then it is identified in the KV store with that address
	// so we know there's one member per pool with a given RUNE address
	// NOTE: We asume that no rune address can have different asset addresses for the same pool
	// as Thornode seems to return an error when a different rune address is added
	addLiquidityQ := `SELECT
		pool,
		COALESCE(MAX(asset_addr), ''),
	` + mpAddLiquidityQFields + `
	FROM stake_events
	WHERE rune_addr = $1
	GROUP BY pool`

	addLiquidityRows, err := db.Query(ctx, addLiquidityQ, runeAddress)
	if err != nil {
		return nil, err
	}
	defer addLiquidityRows.Close()

	memberPoolsMap := make(map[string]MemberPool)

	for addLiquidityRows.Next() {
		memberPool := MemberPool{}
		err := addLiquidityRows.Scan(
			&memberPool.Pool,
			&memberPool.AssetAddress,
			&memberPool.AssetAdded,
			&memberPool.RuneAdded,
			&memberPool.LiquidityUnits,
			&memberPool.DateFirstAdded,
			&memberPool.DateLastAdded,
		)
		if err != nil {
			return nil, err
		}

		memberPool.RuneAddress = runeAddress
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

	// As members need to use the RUNE addresss to withdraw we use it to match each pool
	withdrawQ := `SELECT
		pool,
		` + mpWithdrawQFields + `
	FROM unstake_events
	WHERE from_addr = $1
	GROUP BY pool
	`

	withdrawRows, err := db.Query(ctx, withdrawQ, runeAddress)
	if err != nil {
		return nil, err
	}
	defer withdrawRows.Close()

	for withdrawRows.Next() {
		var pool string
		var assetWithdrawn, runeWithdrawn, unitsWithdrawn int64
		err := withdrawRows.Scan(&pool, &assetWithdrawn, &runeWithdrawn, &unitsWithdrawn)
		if err != nil {
			return nil, err
		}

		memberPool := memberPoolsMap[pool]
		memberPool.AssetWithdrawn = assetWithdrawn
		memberPool.RuneWithdrawn = runeWithdrawn
		memberPool.LiquidityUnits -= unitsWithdrawn

		memberPoolsMap[pool] = memberPool
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

func lpDetailsRune(ctx context.Context, runeAddress, assetAddress, pool string) ([]LPDetail, error) {
	addLiquidityQ := `SELECT
		pool,
	` + lpAddLiquidityQFields + `
	FROM stake_events
	`
	qargs := make([]interface{}, 0)
	if runeAddress == "" {
		addLiquidityQ += `
						WHERE asset_addr = $1
						AND rune_addr IS NULL
						AND pool = $2
						AND (asset_E8 != 0 OR rune_E8 != 0)
						`
		qargs = append(qargs, assetAddress, pool)
	} else if assetAddress == "" {
		addLiquidityQ += `
						WHERE rune_addr = $1
						AND asset_addr IS NULL
						AND pool = $2
						AND (asset_E8 != 0 OR rune_E8 != 0)
						`
		qargs = append(qargs, runeAddress, pool)
	} else {
		addLiquidityQ += `
						WHERE rune_addr = $1
						AND pool = $2
						AND (asset_E8 != 0 OR rune_E8 != 0)
						`
		qargs = append(qargs, runeAddress, pool)
	}

	addLiquidityRows, err := db.Query(ctx, addLiquidityQ, qargs...)
	if err != nil {
		return nil, err
	}
	defer addLiquidityRows.Close()

	lpDetails := make([]LPDetail, 0)
	for addLiquidityRows.Next() {
		lpDetail := LPDetail{}
		err := addLiquidityRows.Scan(
			&lpDetail.Pool,
			&lpDetail.AssetAdded,
			&lpDetail.RuneAdded,
			&lpDetail.LiquidityUnits,
			&lpDetail.Date,
		)
		if err != nil {
			return nil, err
		}
		lpDetails = append(lpDetails, lpDetail)
	}

	withdrawQ := `SELECT
		pool,
		` + lpWithdrawQFields + `
	FROM unstake_events
	WHERE from_addr = $1
	AND pool = $2
	AND (emit_asset_e8 != 0 OR emit_rune_e8 != 0)
	`
	addr := runeAddress
	if addr == "" {
		addr = assetAddress
	}
	withdrawRows, err := db.Query(ctx, withdrawQ, addr, pool)
	if err != nil {
		return nil, err
	}
	defer withdrawRows.Close()

	for withdrawRows.Next() {
		var pool string
		var assetWithdrawn, runeWithdrawn, unitsWithdrawn, date int64
		err := withdrawRows.Scan(&pool, &assetWithdrawn, &runeWithdrawn, &unitsWithdrawn, &date)
		if err != nil {
			return nil, err
		}
		lpDetail := LPDetail{
			LiquidityUnits: -unitsWithdrawn,
			RuneWithdrawn:  runeWithdrawn,
			AssetWithdrawn: assetWithdrawn,
			Date:           date,
		}
		lpDetails = append(lpDetails, lpDetail)
	}
	return lpDetails, nil
}

func getBlockHeight(ctx context.Context, lpDetails []LPDetail) ([]LPDetail, error) {
	dates := make([]int64, 0)
	for _, lp := range lpDetails {
		dates = append(dates, lp.Date)
	}
	datesStr := ""
	for _, d := range dates {
		if len(datesStr) != 0 {
			datesStr += ","
		}
		datesStr += strconv.FormatInt(d, 10)
	}
	blockInfoQ := fmt.Sprintf(`SELECT  height,timestamp
				FROM   block_log
				WHERE  timestamp in (%s) `, datesStr)
	blockInfoRows, err := db.Query(ctx, blockInfoQ)
	if err != nil {
		return nil, err
	}
	defer blockInfoRows.Close()
	var height, timestamp int64
	for blockInfoRows.Next() {
		err := blockInfoRows.Scan(&height, &timestamp)
		if err != nil {
			return nil, err
		}
		for i, lp := range lpDetails {
			if lp.Date == timestamp {
				lpDetails[i].Height = height
			}
		}
	}
	return lpDetails, nil
}

func poolInfo(ctx context.Context, pool string, lpDetails []LPDetail) ([]LPDetail, error) {
	dates := make([]int64, 0)
	for _, lp := range lpDetails {
		dates = append(dates, int64(float64(lp.Date)/(60*5))*(60*5)*(1e+9))
	}
	datesStr := ""
	for _, d := range dates {
		if len(datesStr) != 0 {
			datesStr += ","
		}
		datesStr += strconv.FormatInt(d, 10)
	}
	poolInfoQ := fmt.Sprintf(`SELECT  pool,aggregate_timestamp,priceusd,asset_E8,rune_E8,units
				FROM   midgard_agg.pool_depths_5min
				WHERE  pool = $1
					   AND aggregate_timestamp in (%s) `, datesStr)

	poolInfoRows, err := db.Query(ctx, poolInfoQ, pool)
	if err != nil {
		return nil, err
	}
	defer poolInfoRows.Close()
	var poolInfo PoolInfo
	poolInfos := make([]PoolInfo, 0)
	for poolInfoRows.Next() {
		err := poolInfoRows.Scan(&poolInfo.Pool, &poolInfo.Date, &poolInfo.PriceUsed, &poolInfo.AssetE8, &poolInfo.RuneE8, &poolInfo.Unit)
		if err != nil {
			return nil, err
		}
		poolInfos = append(poolInfos, poolInfo)
	}
	for i, poolDetail := range lpDetails {
		for _, poolInfo := range poolInfos {
			if int64(float64(poolDetail.Date/(60*5))*(60*5)*(1e+9)) == poolInfo.Date {
				lpDetails[i].AssetPriceUsd = poolInfo.PriceUsed
				lpDetails[i].RunePriceUsd = lpDetails[i].AssetPriceUsd / (float64(poolInfo.RuneE8) / float64(poolInfo.AssetE8))
				lpDetails[i].AssetDepth = poolInfo.AssetE8
				lpDetails[i].RuneDepth = poolInfo.RuneE8
				//commented for poolUnit Hotfix
				//lpDetails[i].PoolUnit = poolInfo.Unit
			}
		}
	}
	return lpDetails, nil
}

func memberDetailsAsset(ctx context.Context, assetAddress string) (MemberPools, error) {
	// Get all the rune addresses the asset address is paired with
	addressesQ := `SELECT
		se.pool,
		COALESCE(se.rune_addr, '') as pair_rune_addr
	FROM stake_events AS se
	WHERE se.asset_addr = $1
	GROUP BY pool, pair_rune_addr
	`

	addressesRows, err := db.Query(ctx, addressesQ, assetAddress)
	if err != nil {
		return nil, err
	}
	defer addressesRows.Close()

	var memberPools MemberPools
	for addressesRows.Next() {
		memberPool := MemberPool{AssetAddress: assetAddress}
		err := addressesRows.Scan(&memberPool.Pool, &memberPool.RuneAddress)

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

		addLiquidityQ := `SELECT ` + mpAddLiquidityQFields + `FROM stake_events ` + whereAddLiquidityAddresses + ` AND pool = $2`

		addLiquidityRow, err := db.Query(ctx, addLiquidityQ, queryAddress, memberPool.Pool)
		if err != nil {
			return nil, err
		}
		defer addLiquidityRow.Close()
		if addLiquidityRow.Next() {
			err := addLiquidityRow.Scan(&memberPool.AssetAdded, &memberPool.RuneAdded, &memberPool.LiquidityUnits, &memberPool.DateFirstAdded, &memberPool.DateLastAdded)
			if err != nil {
				return nil, err
			}
		}

		// free up connection early
		addLiquidityRow.Close()

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

		pendingLiquidityRow.Close()

		withdrawQ := `SELECT ` + mpWithdrawQFields + ` FROM unstake_events WHERE from_addr=$1 AND pool=$2`
		withdrawRow, err := db.Query(ctx, withdrawQ, queryAddress, memberPool.Pool)
		if err != nil {
			return nil, err
		}
		defer withdrawRow.Close()
		if withdrawRow.Next() {
			var unitsWithdrawn int64
			err = withdrawRow.Scan(&memberPool.AssetWithdrawn, &memberPool.RuneWithdrawn, &unitsWithdrawn)
			if err != nil {
				return nil, err
			}
			memberPool.LiquidityUnits -= unitsWithdrawn
		}

		if memberPool.LiquidityUnits > 0 {
			memberPools = append(memberPools, memberPool)
		}

		withdrawRow.Close()
	}

	return memberPools, nil
}
