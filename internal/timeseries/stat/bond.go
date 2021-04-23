package stat

import (
	"context"

	"gitlab.com/thorchain/midgard/internal/db"
)

func GetTotalBond(ctx context.Context) (int64, error) {
	bondPaidQ := `
		SELECT
		COALESCE(SUM(E8),0)
		FROM bond_events
		WHERE bond_type = 'bond_paid' OR bond_type = 'bond_reward';
	`
	bondPaidRows, err := db.Query(ctx, bondPaidQ)
	if err != nil {
		return 0, err
	}
	defer bondPaidRows.Close()

	bondReturnedQ := `
		SELECT
		COALESCE(SUM(E8),0)
		FROM bond_events
		WHERE bond_type = 'bond_returned' OR bond_type = 'bond_cost';
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
