package stat

func PoolStatusLookup(pool string) (string, error) {
	const q = `SELECT COALESCE(last(status, block_timestamp), '') FROM pool_events WHERE asset = $1`

	rows, err := DBQuery(q, pool)
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
