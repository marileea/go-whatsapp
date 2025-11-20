package postgres

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DBConfig struct {
	URI         string
	MaxConns    int
	MinConns    int
	MaxIdleTime string
	MaxLifetime string
}

func NewDB(config *DBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", config.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxConns)
	db.SetMaxIdleConns(config.MinConns)

	if config.MaxIdleTime != "" {
		maxIdleTime, err := time.ParseDuration(config.MaxIdleTime)
		if err != nil {
			return nil, fmt.Errorf("invalid max idle time: %w", err)
		}
		db.SetConnMaxIdleTime(maxIdleTime)
	}

	if config.MaxLifetime != "" {
		maxLifetime, err := time.ParseDuration(config.MaxLifetime)
		if err != nil {
			return nil, fmt.Errorf("invalid max lifetime: %w", err)
		}
		db.SetConnMaxLifetime(maxLifetime)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
