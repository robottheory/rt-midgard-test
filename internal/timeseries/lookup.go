package timeseries

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"gitlab.com/thorchain/midgard/chain/notinchain"
)

// ErrBeyondLast denies a request into the future. 💫
var errBeyondLast = errors.New("cannot resolve beyond the last block (timestamp)")

// LastChurnHeight gets the latest block where a vault was activated
func LastChurnHeight(ctx context.Context) (int64, error) {
	q := `SELECT bl.height
	FROM active_vault_events av
	INNER JOIN block_log bl ON av.block_timestamp = bl.timestamp
	ORDER BY av.block_timestamp DESC LIMIT 1;
	`
	var lastChurnHeight int64
	err := QueryOneValue(&lastChurnHeight, ctx, q)
	if err != nil {
		return 0, err
	}
	return lastChurnHeight, nil
}

// Pools gets all asset identifiers for a given point in time.
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func Pools(ctx context.Context, moment time.Time) ([]string, error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return nil, errBeyondLast
	}

	const q = "SELECT pool FROM stake_events WHERE block_timestamp <= $1 GROUP BY pool"
	rows, err := DBQuery(ctx, q, moment.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return pools, err
		}
		pools = append(pools, s)
	}
	return pools, rows.Err()
}

type PoolWithDateCreated struct {
	Asset       string
	DateCreated int64
}

// PoolsWithDateCreated return pools among with the date of the first recorded stake for
// the pool
func PoolsWithDateCreated(ctx context.Context) ([]PoolWithDateCreated, error) {
	q := `SELECT pool, COALESCE(min(block_timestamp), 0) FROM stake_events GROUP BY pool`

	rows, err := DBQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []PoolWithDateCreated
	for rows.Next() {
		var pool PoolWithDateCreated

		err := rows.Scan(&pool.Asset, &pool.DateCreated)
		if err != nil {
			return nil, err
		}

		pools = append(pools, pool)
	}

	return pools, nil
}

// Returns pool->status.
// status is lowercase
func GetPoolsStatuses(ctx context.Context) (map[string]string, error) {
	const q = "SELECT asset, LAST(status, block_timestamp) AS status FROM pool_events GROUP BY asset"

	rows, err := DBQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := map[string]string{}
	for rows.Next() {
		var pool, status string

		err := rows.Scan(&pool, &status)
		status = strings.ToLower(status)
		if err != nil {
			return nil, err
		}

		ret[pool] = status
	}
	return ret, nil
}

// PoolStatus gets the label for a given point in time.
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func PoolStatus(ctx context.Context, pool string, moment time.Time) (string, error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return "", errBeyondLast
	}
	const q = "SELECT COALESCE(last(status, block_timestamp), '') FROM pool_events WHERE block_timestamp <= $2 AND asset = $1"
	rows, err := DBQuery(ctx, q, pool, moment.UnixNano())
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var status string
	if rows.Next() {
		if err := rows.Scan(&status); err != nil {
			return "", err
		}
	}

	if status == "" {
		status = "Bootstrap"
	}
	return status, rows.Err()
}

// PoolUnits gets net stake units in pool
func PoolUnits(ctx context.Context, pool string) (int64, error) {
	q := `SELECT (
		(SELECT COALESCE(SUM(stake_units), 0) FROM stake_events WHERE pool = $1) -
		(SELECT COALESCE(SUM(stake_units), 0) FROM unstake_events WHERE pool = $1)
	)`

	var units int64
	err := QueryOneValue(&units, ctx, q, pool)

	if err != nil {
		return 0, err
	}

	return units, nil
}

// PoolTotalIncome gets sum of liquidity fees and block rewards for a given pool and time interval
func PoolTotalIncome(ctx context.Context, pool string, from time.Time, to time.Time) (int64, error) {
	liquidityFeeQ := `SELECT COALESCE(SUM(liq_fee_in_rune_E8), 0)
	FROM swap_events
	WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp <= $3
	`
	var liquidityFees int64
	err := QueryOneValue(&liquidityFees, ctx, liquidityFeeQ, pool, from.UnixNano(), to.UnixNano())
	if err != nil {
		return 0, err
	}

	blockRewardsQ := `SELECT COALESCE(SUM(rune_E8), 0)
	FROM rewards_event_entries
	WHERE pool = $1 AND block_timestamp >= $2 AND block_timestamp <= $3
	`
	var blockRewards int64
	err = QueryOneValue(&blockRewards, ctx, blockRewardsQ, pool, from.UnixNano(), to.UnixNano())
	if err != nil {
		return 0, err
	}

	return liquidityFees + blockRewards, nil
}

// TotalLiquidityFeesRune gets sum of liquidity fees in Rune for a given time interval
func TotalLiquidityFeesRune(ctx context.Context, from time.Time, to time.Time) (int64, error) {
	liquidityFeeQ := `SELECT COALESCE(SUM(liq_fee_in_rune_E8), 0)
	FROM swap_events
	WHERE block_timestamp >= $1 AND block_timestamp <= $2
	`
	var liquidityFees int64
	err := QueryOneValue(&liquidityFees, ctx, liquidityFeeQ, from.UnixNano(), to.UnixNano())
	if err != nil {
		return 0, err
	}

	return liquidityFees, nil
}

// MemberAddrs gets all member known addresses for a given point in time.
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func MemberAddrs(ctx context.Context) (addrs []string, err error) {
	const q = "SELECT DISTINCT rune_addr FROM stake_events"
	rows, err := DBQuery(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	addrs = make([]string, 0, 1024)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return addrs, err
		}
		addrs = append(addrs, s)
	}
	return addrs, rows.Err()
}

// Mimir gets all values for a given point in time.
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func Mimir(ctx context.Context, moment time.Time) (map[string]string, error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return nil, errBeyondLast
	}

	// could optimise by only fetching latest
	const q = "SELECT name, value FROM set_mimir_event_entries WHERE block_timestamp <= $1"
	rows, err := DBQuery(ctx, q, moment.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("mimir lookup: %w", err)
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var name, value string
		err := rows.Scan(&name, &value)
		if err != nil {
			return m, fmt.Errorf("mimir retrieve: %w", err)
		}
		m[name] = value
	}
	return m, rows.Err()
}

//  Get value from Mimir overrides or from the Thorchain constants.
func GetLastConstantValue(ctx context.Context, key string) (int64, error) {
	// TODO(elfedy): This looks at the last time the mimir value was set. This may not be
	// the latest value (i.e: Does Thorchain send an event with the value in constants if mimir
	// override is unset?). The logic behind this needs to be investigated further.
	q := `SELECT CAST (value AS INTEGER)
	FROM set_mimir_events
	WHERE key ILIKE $1 
	ORDER BY block_timestamp DESC
	LIMIT 1`
	rows, err := DBQuery(ctx, q, key)
	defer rows.Close()
	if err != nil {
		return 0, err
	}
	// Use mimir value if there is one
	var result int64
	if rows.Next() {
		err := rows.Scan(&result)
		if err != nil {
			return 0, err
		}
	} else {
		constants, err := notinchain.ConstantsLookup()

		if err != nil {
			return 0, err
		}
		var ok bool
		result, ok = constants.Int64Values[key]
		if !ok {
			return 0, fmt.Errorf("Key %q not found in constants\n", key)
		}
	}
	return result, nil
}

// StatusPerNode gets the labels for a given point in time.
// New nodes have the empty string (for no confirmed status).
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func StatusPerNode(ctx context.Context, moment time.Time) (map[string]string, error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return nil, errBeyondLast
	}

	m, err := newNodes(ctx, moment)
	if err != nil {
		return nil, err
	}

	// could optimise by only fetching latest
	const q = "SELECT node_addr, current FROM update_node_account_status_events WHERE block_timestamp <= $1"
	rows, err := DBQuery(ctx, q, moment.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("status per node lookup: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var node, status string
		err := rows.Scan(&node, &status)
		if err != nil {
			return m, fmt.Errorf("status per node retrieve: %w", err)
		}
		m[node] = status
	}
	return m, rows.Err()
}

func newNodes(ctx context.Context, moment time.Time) (map[string]string, error) {
	// could optimise by only fetching latest
	const q = "SELECT node_addr FROM new_node_events WHERE block_timestamp <= $1"
	rows, err := DBQuery(ctx, q, moment.UnixNano())
	if err != nil {
		return nil, fmt.Errorf("new node lookup: %w", err)
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var node string
		err := rows.Scan(&node)
		if err != nil {
			return m, fmt.Errorf("new node retrieve: %w", err)
		}
		m[node] = ""
	}
	return m, rows.Err()
}

// NodesSecpAndEd returs the public keys mapped to their respective addresses.
func NodesSecpAndEd(ctx context.Context, t time.Time) (secp256k1Addrs, ed25519Addrs map[string]string, err error) {
	const q = `SELECT node_addr, secp256k1, ed25519
FROM set_node_keys_events
WHERE block_timestamp <= $1`

	rows, err := DBQuery(ctx, q, t.UnixNano())
	if err != nil {
		return nil, nil, fmt.Errorf("node addr lookup: %w", err)
	}
	defer rows.Close()

	secp256k1Addrs = make(map[string]string)
	ed25519Addrs = make(map[string]string)
	for rows.Next() {
		var addr, secp, ed string
		if err := rows.Scan(&addr, &secp, &ed); err != nil {
			return nil, nil, fmt.Errorf("node addr resolve: %w", err)
		}
		if current, ok := secp256k1Addrs[secp]; ok && current != addr {
			log.Printf("secp256k1 key %q used by node address %q and %q", secp, current, addr)
		}
		secp256k1Addrs[secp] = addr
		if current, ok := ed25519Addrs[ed]; ok && current != addr {
			log.Printf("Ed25519 key %q used by node address %q and %q", ed, current, addr)
		}
		ed25519Addrs[secp] = addr
	}
	return
}
