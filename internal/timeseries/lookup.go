package timeseries

import (
	"errors"
	"fmt"
	"time"
)

// ErrBeyondLast denies a request into the future. 💫
var errBeyondLast = errors.New("cannot resolve beyond the last block (timestamp)")

// Pools gets all asset identifiers for a given point in time.
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func Pools(moment time.Time) ([]string, error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return nil, errBeyondLast
	}

	const q = "SELECT pool FROM stake_events WHERE block_timestamp <= $1 GROUP BY pool"
	rows, err := DBQuery(q, moment.UnixNano())
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

// PoolStatus gets the label for a given point in time.
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func PoolStatus(pool string, moment time.Time) (string, error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return "", errBeyondLast
	}

	const q = "SELECT COALESCE(last(status, block_timestamp), '') FROM pool_events WHERE block_timestamp <= $2 AND asset = $1"
	rows, err := DBQuery(q, pool, moment.UnixNano())
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
	return status, rows.Err()
}

// StakeAddrs gets all known addresses for a given point in time.
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func StakeAddrs(moment time.Time) (addrs []string, err error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return nil, errBeyondLast
	}

	const q = "SELECT rune_addr FROM stake_events WHERE block_timestamp <= $1 GROUP BY rune_addr"
	rows, err := DBQuery(q, moment.UnixNano())
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
func Mimir(moment time.Time) (map[string]string, error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return nil, errBeyondLast
	}

	// could optimise by only fetching latest
	const q = "SELECT name, value FROM set_mimir_event_entries WHERE block_timestamp <= $1"
	rows, err := DBQuery(q, moment.UnixNano())
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

// StatusPerNode gets the labels for a given point in time.
// New nodes have the empty string (for no confirmed status).
// A zero moment defaults to the latest available.
// Requests beyond the last block cause an error.
func StatusPerNode(moment time.Time) (map[string]string, error) {
	_, timestamp, _ := LastBlock()
	if moment.IsZero() {
		moment = timestamp
	} else if timestamp.Before(moment) {
		return nil, errBeyondLast
	}

	m, err := newNodes(moment)
	if err != nil {
		return nil, err
	}

	// could optimise by only fetching latest
	const q = "SELECT node_addr, current FROM update_node_account_status_events WHERE block_timestamp <= $1"
	rows, err := DBQuery(q, moment.UnixNano())
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

func newNodes(moment time.Time) (map[string]string, error) {
	// could optimise by only fetching latest
	const q = "SELECT node_addr FROM new_node_events WHERE block_timestamp <= $1"
	rows, err := DBQuery(q, moment.UnixNano())
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
