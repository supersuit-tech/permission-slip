package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// GetActionDataWindowParams returns the declared window parameter pair for an action.
func GetActionDataWindowParams(ctx context.Context, d DBTX, actionType string) (*DataWindowParams, error) {
	var raw []byte
	err := d.QueryRow(ctx,
		`SELECT data_window FROM connector_actions WHERE action_type = $1`,
		actionType,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var params DataWindowParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, err
	}
	if params.StartParam == "" || params.EndParam == "" {
		return nil, nil
	}
	return &params, nil
}
