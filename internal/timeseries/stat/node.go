package stat

import "time"

type NodeKeys struct {
	NodeAddr           string
	Secp256k1          string
	Ed25519            string
	ValidatorConsensus string
}

func NodeKeysLookup(t time.Time) ([]NodeKeys, error) {
	const q = `SELECT node_addr, secp256k1, ed25519, validator_consensus
FROM set_node_keys_events
WHERE block_timestamp <= $1`

	rows, err := DBQuery(q, t.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var a []NodeKeys
	for rows.Next() {
		var r NodeKeys
		if err := rows.Scan(&r.NodeAddr, &r.Secp256k1, &r.Ed25519, &r.ValidatorConsensus); err != nil {
			return a, err
		}
		a = append(a, r)
	}
	return a, rows.Err()
}
