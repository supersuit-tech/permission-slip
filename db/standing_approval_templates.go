package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// StandingApprovalTemplate represents a row from the standing_approval_templates table.
// Templates are system-level presets that users can apply to create standing approvals.
type StandingApprovalTemplate struct {
	ID           string
	ConnectorID  string
	ActionType   string
	Name         string
	Description  *string
	Constraints  []byte // raw JSONB
	DurationDays *int
	CreatedAt    time.Time
}

// MaxStandingApprovalTemplateListSize is the maximum number of templates returned by list queries.
const MaxStandingApprovalTemplateListSize = 100

func scanStandingApprovalTemplate(row rowScanner) (*StandingApprovalTemplate, error) {
	var t StandingApprovalTemplate
	var durationDays sql.NullInt64
	var createdAt sql.NullString
	err := row.Scan(
		&t.ID, &t.ConnectorID, &t.ActionType, &t.Name,
		&t.Description, &t.Constraints, &durationDays, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	if durationDays.Valid {
		d := int(durationDays.Int64)
		t.DurationDays = &d
	}
	var err2 error
	t.CreatedAt, err2 = sqliteTimeRequired(createdAt)
	if err2 != nil {
		return nil, err2
	}
	return &t, nil
}

// ListStandingApprovalTemplatesByConnector returns all standing approval templates
// for the given connector, ordered by action_type then name.
func ListStandingApprovalTemplatesByConnector(ctx context.Context, db DBTX, connectorID string) ([]StandingApprovalTemplate, error) {
	rows, err := db.Query(ctx,
		`SELECT id, connector_id, action_type, name, description, constraints, duration_days, created_at
		 FROM standing_approval_templates
		 WHERE connector_id = $1
		 ORDER BY action_type, name
		 LIMIT $2`,
		connectorID, MaxStandingApprovalTemplateListSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []StandingApprovalTemplate
	for rows.Next() {
		tpl, err := scanStandingApprovalTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *tpl)
	}
	return templates, rows.Err()
}

// GetStandingApprovalTemplatesByIDs returns templates matching the given IDs.
func GetStandingApprovalTemplatesByIDs(ctx context.Context, db DBTX, ids []string) ([]StandingApprovalTemplate, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := db.Query(ctx,
		`SELECT id, connector_id, action_type, name, description, constraints, duration_days, created_at
		 FROM standing_approval_templates
		 WHERE id IN (`+InPlaceholders(1, len(ids))+`)`,
		StringsToArgs(ids)...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []StandingApprovalTemplate
	for rows.Next() {
		tpl, err := scanStandingApprovalTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *tpl)
	}
	return templates, rows.Err()
}

// GetStandingApprovalTemplateByID returns a single template by ID, or nil if not found.
func GetStandingApprovalTemplateByID(ctx context.Context, db DBTX, id string) (*StandingApprovalTemplate, error) {
	row := db.QueryRow(ctx,
		`SELECT id, connector_id, action_type, name, description, constraints, duration_days, created_at
		 FROM standing_approval_templates
		 WHERE id = $1`,
		id,
	)
	tpl, err := scanStandingApprovalTemplate(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return tpl, nil
}
