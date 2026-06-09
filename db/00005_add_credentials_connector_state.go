package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upAddCredentialsConnectorState, downAddCredentialsConnectorState)
}

func upAddCredentialsConnectorState(ctx context.Context, tx *sql.Tx) error {
	return addCredentialsColumnIfMissing(ctx, tx, "connector_state",
		`ALTER TABLE credentials ADD COLUMN connector_state TEXT NOT NULL DEFAULT '{}'`)
}

func downAddCredentialsConnectorState(ctx context.Context, tx *sql.Tx) error {
	// SQLite cannot drop columns without a table rebuild; leave in place.
	return nil
}

func addCredentialsColumnIfMissing(ctx context.Context, tx *sql.Tx, column, alterSQL string) error {
	exists, err := credentialsColumnExists(ctx, tx, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if _, err := tx.ExecContext(ctx, alterSQL); err != nil {
		return fmt.Errorf("add credentials column %s: %w", column, err)
	}
	return nil
}

func credentialsColumnExists(ctx context.Context, tx *sql.Tx, column string) (bool, error) {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM pragma_table_info('credentials')`)
	if err != nil {
		return false, fmt.Errorf("inspect credentials columns: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
