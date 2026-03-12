package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetActionParametersSchema returns the parameters_schema JSON for the given
// action type. Returns nil if the action type doesn't exist or has no schema.
func GetActionParametersSchema(ctx context.Context, db DBTX, actionType string) ([]byte, error) {
	var schema []byte
	err := db.QueryRow(ctx,
		`SELECT parameters_schema FROM connector_actions WHERE action_type = $1`,
		actionType,
	).Scan(&schema)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return schema, nil
}
