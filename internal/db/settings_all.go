package db

// AllSettings returns every key/value pair in the settings table. Used by
// the debug bundle to emit a scrubbed snapshot. Empty map on error so
// callers do not need to special-case nil.
func (d *DB) AllSettings() (map[string]string, error) {
	rows, err := d.Query("SELECT key, value FROM settings ORDER BY key")
	if err != nil {
		return map[string]string{}, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		out[k] = v
	}
	return out, rows.Err()
}
