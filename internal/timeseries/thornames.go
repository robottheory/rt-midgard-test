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

	q := `
		WITH gp_names AS 
		(
			SELECT *, ROW_NUMBER() OVER (PARTITION BY name, chain ORDER BY block_timestamp DESC) as row_number 
			FROM thorname_change_events
		) 
		SELECT chain, address, c.expire as expire, c.owner as owner
		FROM gp_names, 
		(	
			SELECT expire, owner FROM thorname_change_events WHERE name = $1 AND expire > $2 
			ORDER BY block_timestamp DESC LIMIT 1
		) as c
		WHERE 
				row_number = 1 
			AND 
				name = $1
			AND
				c.expire > $2
		ORDER BY
			block_timestamp DESC
	`

	rows, err := db.Query(ctx, q, name, currentHeight)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var entry THORNameEntry
		if err := rows.Scan(&entry.Chain, &entry.Address, &tName.Expire, &tName.Owner); err != nil {
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
	currentHeight, _, _ := LastBlock()

	q := `
		WITH gp_names AS 
		(SELECT *, ROW_NUMBER() OVER (PARTITION BY name, chain ORDER BY block_timestamp DESC) as row_number 
		FROM thorname_change_events) 
		SELECT DISTINCT on (name) name 
		FROM gp_names 
		WHERE 
			row_number = 1 
		AND 
			address = $1
		AND
			expire > $2
	`

	rows, err := db.Query(ctx, q, addr, currentHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	return
}

func GetTHORNamesByOwnerAddress(ctx context.Context, addr *string) (names []string, err error) {
	currentHeight, _, _ := LastBlock()

	q := `
		WITH gp_names AS 
			(SELECT *, ROW_NUMBER() OVER (PARTITION BY name, chain ORDER BY block_timestamp DESC) as row_number 
		FROM thorname_change_events) 
		SELECT DISTINCT on (name) name 
		FROM gp_names 
		WHERE 
			row_number = 1 
		AND 
			owner = $1
		AND
			expire > $2
	`

	rows, err := db.Query(ctx, q, addr, currentHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	return
}
