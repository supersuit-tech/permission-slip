package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ActionSchema holds the connector ID and parameters schema for an action type.
type ActionSchema struct {
	ConnectorID string
	Schema      []byte
}

// GetActionParametersSchema returns the connector ID and parameters_schema for
// the given action type. Returns nil if the action type doesn't exist or has
// no schema.
func GetActionParametersSchema(ctx context.Context, db DBTX, actionType string) (*ActionSchema, error) {
	var result ActionSchema
	err := db.QueryRow(ctx,
		`SELECT connector_id, parameters_schema FROM connector_actions WHERE action_type = $1`,
		actionType,
	).Scan(&result.ConnectorID, &result.Schema)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}
