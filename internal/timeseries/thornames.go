package timeseries

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

type THORNameEntry struct {
	Chain   string
	Address string
}

type THORName struct {
	Owner   string
	Expire  int64
	Entries []THORNameEntry
}

func GetTHORName(ctx context.Context, name *string) (tName THORName, err error) {
	currentHeight, _, _ := LastBlock()

	// Expiration of THORName is tracked only by the "THOR" record. All other
	// chains follow suit with the status of this "root" record.
	q := `
		SELECT
			expire, owner
		FROM thorname_change_events
		WHERE
			expire > $1 AND name = $2
		ORDER BY
			block_timestamp DESC
		LIMIT 1
	`

	rows, err := db.Query(ctx, q, currentHeight, name)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&tName.Expire, &tName.Owner); err != nil {
			return tName, err
		}
		break
	}

	// check if we found a name
	if tName.Owner == "" {
		return
	}

	q = `
		SELECT
			DISTINCT on (chain) chain, address
		FROM thorname_change_events
		WHERE
			name = $1
		ORDER BY
			chain, block_timestamp DESC
	`

	rows, err = db.Query(ctx, q, name)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var entry THORNameEntry
		if err := rows.Scan(&entry.Chain, &entry.Address); err != nil {
			return tName, err
		}
		tName.Entries = append(tName.Entries, entry)
	}

	return
}

// NOTE: there is probably a pure-postrgres means of doing this, which would be
// more performant. If we find that the performance of this query to be too
// slow, can try that. I don't imagine it being much of a problem since people
// aren't going to associate their address with 100's of thornames
func GetTHORNamesByAddress(ctx context.Context, addr *string) (names []string, err error) {
	q := `
		SELECT
			DISTINCT on (name) name
		FROM thorname_change_events
		WHERE
			address = $1
	`

	rows, err := db.Query(ctx, q, addr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		// validate the address is associated with the current record of THORname
		tName, err := GetTHORName(ctx, &name)
		if err != nil {
			continue
		}
		for _, e := range tName.Entries {
			if e.Address == *addr {
				names = append(names, name)
				break
			}
		}
	}

	return
}

func GetTHORNamesByOwnerAddress(ctx context.Context, addr *string) (names []string, err error) {
	// TODO(HooriRn): Add a new test for covering these cases then fix potential issues, or add a
	//   note if it's hard to fix:
	//     * [x] owner and and address is different
	//     * [x] thorname owner expired, should return empty
	//     * [x] thorname event didn't expire yet, but there was a newer event with a different owner,
	//       should return empty.
	q := `
		SELECT
			DISTINCT on (name) name
		FROM thorname_change_events
		WHERE
			owner = $1
	`

	rows, err := db.Query(ctx, q, addr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		tName, err := GetTHORName(ctx, &name)
		if err != nil && tName.Owner == "" {
			continue
		}

		if tName.Owner == *addr {
			names = append(names, name)
		}
	}

	return
}
