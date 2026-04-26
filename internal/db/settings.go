package db

func (d *DB) GetSetting(key string) (string, error) {
	var val string
	err := d.QueryRow("SELECT value FROM settings WHERE key=?", key).Scan(&val)
	return val, err
}

func (d *DB) SetSetting(key, value string) error {
	_, err := d.Exec("INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, value)
	return err
}
