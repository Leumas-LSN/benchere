package config

import "os"

type Config struct {
	Port   string
	DBPath string
	Debug  bool
}

func Load() Config {
	debug := os.Getenv("BENCHERE_DEBUG") == "true"
	port := os.Getenv("BENCHERE_PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("BENCHERE_DB")
	if dbPath == "" {
		dbPath = "/opt/benchere/benchere.db"
	}
	return Config{Port: port, DBPath: dbPath, Debug: debug}
}
