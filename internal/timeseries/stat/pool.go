package stat

func PoolStatusLookup(pool string) (string, error) {
	var status string
	const q = `SELECT last(status, block_timestamp) FROM pool_events WHERE asset = $1`

	rows, err := DBQuery(q, pool)
	if err != nil {
		return status, err
	}
	defer rows.Close()

	if !rows.Next() {
		return status, rows.Err()
	}

	if err := rows.Scan(&status); err != nil {
		return status, err
	}

	// TODO (manolodewiner) Query THORChain if we haven't received any pool event for the specified pool --> usecase.go

	return status, rows.Err()
}
