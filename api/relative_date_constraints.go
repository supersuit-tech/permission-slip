package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// applyStandingApprovalRelativeDates resolves relative date tokens in standing
// approval constraints and injects/clamps the connector's datetime parameters.
func applyStandingApprovalRelativeDates(
	ctx context.Context,
	d db.DBTX,
	actionType string,
	constraints json.RawMessage,
	params json.RawMessage,
	now time.Time,
) (json.RawMessage, error) {
	tokens := db.CollectRelativeDateConstraintTokens(constraints)
	if len(tokens) == 0 {
		return params, nil
	}

	schema, err := db.GetActionParametersSchema(ctx, d, actionType)
	if err != nil {
		return nil, err
	}
	if schema == nil || len(schema.Schema) == 0 {
		return nil, fmt.Errorf("relative date constraints require action schema for %q", actionType)
	}

	fields, err := db.ParseActionSchemaDateTimeFields(schema.Schema)
	if err != nil {
		return nil, err
	}
	for field, token := range tokens {
		if _, ok := fields[field]; !ok {
			return nil, &db.ConfigValidationError{
				Parameter: field,
				Reason:    fmt.Sprintf("relative date token %q is only valid on date or date-time parameters", token),
			}
		}
	}

	// User profile timezone is not stored yet; resolve in UTC until profile TZ lands.
	return db.ApplyRelativeDateConstraintsToParams(params, constraints, fields, now, time.UTC)
}
