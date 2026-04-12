package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ActionConfigTemplate represents a row from the action_config_templates table.
// Templates are system-level presets that users can apply when creating action
// configurations, pre-filling the name, description, and parameter constraints.
type ActionConfigTemplate struct {
	ID                    string
	ConnectorID           string
	ActionType            string
	Name                  string
	Description           *string
	Parameters            []byte // raw JSONB
	StandingApprovalSpec  []byte // raw JSONB; nil = no bundled standing approval
	CreatedAt             time.Time
}

// MaxTemplateListSize is the maximum number of templates returned by list queries.
const MaxTemplateListSize = 100

// ListTemplatesByConnector returns all action configuration templates for the
// given connector, ordered by action_type then name. Results are capped at
// MaxTemplateListSize.
func ListTemplatesByConnector(ctx context.Context, db DBTX, connectorID string) ([]ActionConfigTemplate, error) {
	rows, err := db.Query(ctx,
		`SELECT id, connector_id, action_type, name, description, parameters, standing_approval_spec, created_at
		 FROM action_config_templates
		 WHERE connector_id = $1
		 ORDER BY action_type, name
		 LIMIT $2`,
		connectorID, MaxTemplateListSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []ActionConfigTemplate
	for rows.Next() {
		var t ActionConfigTemplate
		if err := rows.Scan(&t.ID, &t.ConnectorID, &t.ActionType, &t.Name,
			&t.Description, &t.Parameters, &t.StandingApprovalSpec, &t.CreatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// GetActionConfigTemplateByID returns the action configuration template with the
// given ID, or nil if not found.
func GetActionConfigTemplateByID(ctx context.Context, db DBTX, templateID string) (*ActionConfigTemplate, error) {
	row := db.QueryRow(ctx,
		`SELECT id, connector_id, action_type, name, description, parameters, standing_approval_spec, created_at
		 FROM action_config_templates
		 WHERE id = $1`,
		templateID,
	)
	var t ActionConfigTemplate
	if err := row.Scan(&t.ID, &t.ConnectorID, &t.ActionType, &t.Name,
		&t.Description, &t.Parameters, &t.StandingApprovalSpec, &t.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
