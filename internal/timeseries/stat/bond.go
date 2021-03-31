package stat

// Note that these values don't sum up to the actual bonds reported by ThorNode.
// There are changes to the bond not present in the events.
// Possibly remove this file in the future bonds when we have a different plan with bonds.

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

func GetTotalBond(ctx context.Context) (int64, error) {
	bondPaidQ := `
		SELECT
		COALESCE(SUM(asset_E8),0)
		FROM bond_events
		WHERE bond_type = 'bond_paid';
	`
	bondPaidRows, err := db.Query(ctx, bondPaidQ)
	if err != nil {
		return 0, err
	}
	defer bondPaidRows.Close()

	bondReturnedQ := `
		SELECT
		COALESCE(SUM(asset_E8),0)
		FROM bond_events
		WHERE bond_type = 'bond_returned';
	`
	bondReturnedRows, err := db.Query(ctx, bondReturnedQ)
	if err != nil {
		return 0, err
	}
	defer bondReturnedRows.Close()

	// PROCESS DATA
	// Create aggregate variables to be filled with row results

	var totalBond int64

	for bondPaidRows.Next() {
		var x int64
		err := bondPaidRows.Scan(&x)
		if err != nil {
			return 0, err
		}
		totalBond += x
	}

	for bondReturnedRows.Next() {
		var x int64
		err := bondReturnedRows.Scan(&x)
		if err != nil {
			return 0, err
		}
		totalBond -= x
	}

	return totalBond, nil
}
