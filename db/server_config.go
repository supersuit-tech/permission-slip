package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// GetServerConfig returns the value for the given key, or empty string if not found.
func GetServerConfig(ctx context.Context, db DBTX, key string) (string, error) {
	var value string
	err := db.QueryRow(ctx,
		"SELECT value FROM server_config WHERE key = $1",
		key,
	).Scan(&value)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetServerConfig inserts or updates a server config key-value pair.
func SetServerConfig(ctx context.Context, db DBTX, key, value string) error {
	_, err := db.Exec(ctx,
		`INSERT INTO server_config (key, value)
		 VALUES ($1, $2)
		 ON CONFLICT (key)
		 DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		key, value,
	)
	return err
}
