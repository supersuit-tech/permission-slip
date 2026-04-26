package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ConnectorSummary represents a connector with its action types and required credential services.
type ConnectorSummary struct {
	ID                  string
	Name                string
	Description         *string
	Status              string
	LogoSVG             *string
	Actions             []string // action_type values
	RequiredCredentials []string // service values
}

// ConnectorDetail represents a connector with full action details and required credentials.
type ConnectorDetail struct {
	ID                  string
	Name                string
	Description         *string
	Status              string
	LogoSVG             *string
	Actions             []ConnectorAction
	RequiredCredentials []RequiredCredential
}

// ConnectorAction represents a row from the connector_actions table.
type ConnectorAction struct {
	ActionType            string
	OperationType         string // read, write, edit, or delete
	Name                  string
	Description           *string
	RiskLevel             *string
	ParametersSchema      []byte // raw JSONB
	RequiresPaymentMethod bool
	DisplayTemplate       *string
	Preview               []byte // raw JSONB — structured preview layout config
}

// CredentialFieldSpec is one field in a static (api_key/custom) credential schema.
type CredentialFieldSpec struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Secret      bool   `json:"secret"`
	Required    bool   `json:"required"`
	HelpText    string `json:"help_text,omitempty"`
}

// RequiredCredential represents a row from the connector_required_credentials table.
type RequiredCredential struct {
	Service          string
	AuthType         string
	InstructionsURL  *string
	OAuthProvider    *string
	OAuthScopes      []string
	CredentialFields []CredentialFieldSpec // from credential_fields JSONB; empty means UI default single api_key field
}

// ListConnectors returns all connectors with their action types and required credential services.
func ListConnectors(ctx context.Context, db DBTX) ([]ConnectorSummary, error) {
	rows, err := db.Query(ctx, `
		SELECT c.id, c.name, c.description, c.status, c.logo_svg,
		       COALESCE(array_agg(DISTINCT ca.action_type ORDER BY ca.action_type) FILTER (WHERE ca.action_type IS NOT NULL), '{}'),
		       COALESCE(array_agg(DISTINCT crc.service ORDER BY crc.service) FILTER (WHERE crc.service IS NOT NULL), '{}')
		FROM connectors c
		LEFT JOIN connector_actions ca ON ca.connector_id = c.id
		LEFT JOIN connector_required_credentials crc ON crc.connector_id = c.id
		GROUP BY c.id, c.name, c.description, c.status, c.logo_svg
		ORDER BY c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connectors []ConnectorSummary
	for rows.Next() {
		var cs ConnectorSummary
		if err := rows.Scan(&cs.ID, &cs.Name, &cs.Description, &cs.Status, &cs.LogoSVG, &cs.Actions, &cs.RequiredCredentials); err != nil {
			return nil, err
		}
		connectors = append(connectors, cs)
	}
	return connectors, rows.Err()
}

// GetConnectorByID returns a single connector with full action details and required credentials.
// Returns nil if the connector doesn't exist.
func GetConnectorByID(ctx context.Context, db DBTX, connectorID string) (*ConnectorDetail, error) {
	// Fetch the connector row.
	var cd ConnectorDetail
	err := db.QueryRow(ctx,
		`SELECT id, name, description, status, logo_svg FROM connectors WHERE id = $1`,
		connectorID,
	).Scan(&cd.ID, &cd.Name, &cd.Description, &cd.Status, &cd.LogoSVG)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Fetch actions.
	actionRows, err := db.Query(ctx,
		`SELECT action_type, operation_type, name, description, risk_level, parameters_schema, requires_payment_method, display_template, preview
		 FROM connector_actions
		 WHERE connector_id = $1
		 ORDER BY action_type`,
		connectorID,
	)
	if err != nil {
		return nil, err
	}
	defer actionRows.Close()

	for actionRows.Next() {
		var a ConnectorAction
		if err := actionRows.Scan(&a.ActionType, &a.OperationType, &a.Name, &a.Description, &a.RiskLevel, &a.ParametersSchema, &a.RequiresPaymentMethod, &a.DisplayTemplate, &a.Preview); err != nil {
			return nil, err
		}
		cd.Actions = append(cd.Actions, a)
	}
	if err := actionRows.Err(); err != nil {
		return nil, err
	}

	// Fetch required credentials.
	credRows, err := db.Query(ctx,
		`SELECT service, auth_type, instructions_url, oauth_provider, oauth_scopes, COALESCE(credential_fields, '[]'::jsonb)
		 FROM connector_required_credentials
		 WHERE connector_id = $1
		 ORDER BY service`,
		connectorID,
	)
	if err != nil {
		return nil, err
	}
	defer credRows.Close()

	for credRows.Next() {
		var rc RequiredCredential
		var fieldsRaw []byte
		if err := credRows.Scan(&rc.Service, &rc.AuthType, &rc.InstructionsURL, &rc.OAuthProvider, &rc.OAuthScopes, &fieldsRaw); err != nil {
			return nil, err
		}
		if len(fieldsRaw) > 0 && string(fieldsRaw) != "[]" && string(fieldsRaw) != "null" {
			if err := json.Unmarshal(fieldsRaw, &rc.CredentialFields); err != nil {
				return nil, fmt.Errorf("unmarshal credential_fields for %s/%s: %w", rc.Service, rc.AuthType, err)
			}
		}
		cd.RequiredCredentials = append(cd.RequiredCredentials, rc)
	}
	return &cd, credRows.Err()
}

// GetRequiredCredentialsByService returns all connector_required_credentials rows
// matching the given service name (may be multiple connectors if they share a
// service string — callers should handle ambiguity).
func GetRequiredCredentialsByService(ctx context.Context, db DBTX, service string) ([]RequiredCredential, error) {
	rows, err := db.Query(ctx, `
		SELECT service, auth_type, instructions_url, oauth_provider, oauth_scopes, COALESCE(credential_fields, '[]'::jsonb)
		FROM connector_required_credentials
		WHERE service = $1`,
		service,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RequiredCredential
	for rows.Next() {
		var rc RequiredCredential
		var fieldsRaw []byte
		if err := rows.Scan(&rc.Service, &rc.AuthType, &rc.InstructionsURL, &rc.OAuthProvider, &rc.OAuthScopes, &fieldsRaw); err != nil {
			return nil, err
		}
		if len(fieldsRaw) > 0 && string(fieldsRaw) != "[]" && string(fieldsRaw) != "null" {
			if err := json.Unmarshal(fieldsRaw, &rc.CredentialFields); err != nil {
				return nil, fmt.Errorf("unmarshal credential_fields for service %q: %w", service, err)
			}
		}
		out = append(out, rc)
	}
	return out, rows.Err()
}

// GetRequiredServicesByActionType returns the list of static credential services
// (non-OAuth2) required by the connector that owns the given action type.
// OAuth2 services are excluded because they are resolved through the OAuth
// connection path, not through static credential storage.
// Returns an empty slice if the action type has no required credentials.
// Returns nil, nil if the action type is not found in the database.
func GetRequiredServicesByActionType(ctx context.Context, db DBTX, actionType string) ([]string, error) {
	// Use a LEFT JOIN so that a matching action with no required credentials
	// returns a single row with a NULL service (→ empty slice) while a
	// non-matching action type returns zero rows (→ nil, nil).
	// Exclude oauth2 services — those are resolved via resolveOAuthCredentials.
	rows, err := db.Query(ctx, `
		SELECT crc.service
		FROM connector_actions ca
		LEFT JOIN connector_required_credentials crc
		       ON crc.connector_id = ca.connector_id
		          AND crc.auth_type != 'oauth2'
		WHERE ca.action_type = $1
		ORDER BY crc.service`,
		actionType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		services []string
		found    bool
	)
	for rows.Next() {
		found = true
		var s *string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		if s != nil {
			services = append(services, *s)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	if services == nil {
		services = []string{}
	}
	return services, nil
}

// ListConnectorIDs returns the IDs of all connectors in the database.
// Used for startup validation to detect mismatches with code-registered connectors.
func ListConnectorIDs(ctx context.Context, db DBTX) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT id FROM connectors ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeleteConnectorByID removes a connector and dependent rows (actions,
// credentials, templates, agent links) via ON DELETE CASCADE.
// Returns how many rows were deleted (0 if the id was not present).
func DeleteConnectorByID(ctx context.Context, db DBTX, connectorID string) (int64, error) {
	ct, err := db.Exec(ctx, `DELETE FROM connectors WHERE id = $1`, connectorID)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// GetActionRequiresPaymentMethod checks whether the given action type requires
// a payment method. Returns (true, nil) if the action exists and requires payment,
// (false, nil) if it exists but doesn't require payment, and (false, error) if
// the action type is not found or a query error occurs.
func GetActionRequiresPaymentMethod(ctx context.Context, db DBTX, actionType string) (bool, error) {
	var requires bool
	err := db.QueryRow(ctx,
		`SELECT requires_payment_method FROM connector_actions WHERE action_type = $1`,
		actionType,
	).Scan(&requires)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return requires, nil
}

// ExternalConnectorManifest contains the data needed to upsert a connector
// from an external connector's manifest. This is a plain struct (no dependency
// on the connectors package) to keep the db package import-free of connectors/.
type ExternalConnectorManifest struct {
	ID          string
	Name        string
	Description string
	Status      string
	LogoSVG     string
	Actions     []ExternalConnectorAction
	Credentials []ExternalConnectorCredential
	Templates   []ExternalConnectorTemplate
}

// ExternalConnectorAction describes an action from an external connector manifest.
type ExternalConnectorAction struct {
	ActionType            string
	OperationType         string // read, write, edit, or delete
	Name                  string
	Description           string
	RiskLevel             string
	ParametersSchema      []byte // raw JSON
	RequiresPaymentMethod bool
	DisplayTemplate       string
	Preview               []byte // raw JSON — structured preview layout config
}

// ExternalConnectorCredential describes a required credential from an external connector manifest.
type ExternalConnectorCredential struct {
	Service         string
	AuthType        string
	InstructionsURL string
	OAuthProvider   string
	OAuthScopes     []string
	FieldsJSON      []byte // JSON array of CredentialFieldSpec; nil/empty → store as []
}

// ExternalConnectorTemplate describes a configuration template from a connector manifest.
type ExternalConnectorTemplate struct {
	ID                   string
	ActionType           string
	Name                 string
	Description          string
	Parameters           []byte // raw JSON
	StandingApprovalSpec []byte // raw JSON; nil = no bundled standing approval
}

// UpsertConnectorFromManifest inserts or updates a connector and its actions
// and required credentials from an external connector manifest, and removes
// stale actions/credentials no longer present in the manifest. This is called
// on server startup so external connector DB rows stay in sync with manifests.
// The operation is wrapped in a transaction to ensure atomicity.
func UpsertConnectorFromManifest(ctx context.Context, d DBTX, m ExternalConnectorManifest) error {
	tx, owned, err := BeginOrContinue(ctx, d)
	if err != nil {
		return err
	}
	if owned {
		defer func() { _ = RollbackTx(ctx, tx) }()
	}

	// Upsert the connector record.
	// Status is always set by ToDBManifest, but default defensively in case
	// ExternalConnectorManifest is constructed directly by other callers.
	status := m.Status
	if status == "" {
		status = "untested"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO connectors (id, name, description, status, logo_svg)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, status = EXCLUDED.status, logo_svg = EXCLUDED.logo_svg`,
		m.ID, m.Name, nilIfEmpty(m.Description), status, nilIfEmpty(m.LogoSVG))
	if err != nil {
		return err
	}

	// Upsert actions.
	actionTypes := make([]string, 0, len(m.Actions))
	for _, a := range m.Actions {
		actionTypes = append(actionTypes, a.ActionType)
		opType := a.OperationType
		if opType == "" {
			opType = "write"
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO connector_actions (connector_id, action_type, operation_type, name, description, risk_level, parameters_schema, requires_payment_method, display_template, preview)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (connector_id, action_type) DO UPDATE SET
				operation_type = EXCLUDED.operation_type,
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				risk_level = EXCLUDED.risk_level,
				parameters_schema = EXCLUDED.parameters_schema,
				requires_payment_method = EXCLUDED.requires_payment_method,
				display_template = EXCLUDED.display_template,
				preview = EXCLUDED.preview`,
			m.ID, a.ActionType, opType, a.Name, nilIfEmpty(a.Description), nilIfEmpty(a.RiskLevel), nilIfEmptyBytes(a.ParametersSchema), a.RequiresPaymentMethod, nilIfEmpty(a.DisplayTemplate), nilIfEmptyBytes(a.Preview))
		if err != nil {
			return err
		}
	}

	// Remove actions no longer in the manifest.
	if len(actionTypes) > 0 {
		_, err = tx.Exec(ctx,
			`DELETE FROM connector_actions WHERE connector_id = $1 AND action_type != ALL($2)`,
			m.ID, actionTypes)
	} else {
		_, err = tx.Exec(ctx,
			`DELETE FROM connector_actions WHERE connector_id = $1`, m.ID)
	}
	if err != nil {
		return err
	}

	// Upsert required credentials.
	// Track (service, auth_type) pairs so we can remove stale rows afterwards.
	type serviceAuthKey struct{ service, authType string }
	credKeys := make([]serviceAuthKey, 0, len(m.Credentials))
	for _, c := range m.Credentials {
		credKeys = append(credKeys, serviceAuthKey{c.Service, c.AuthType})
		fieldsVal := c.FieldsJSON
		if len(fieldsVal) == 0 {
			fieldsVal = []byte("[]")
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO connector_required_credentials (connector_id, service, auth_type, instructions_url, oauth_provider, oauth_scopes, credential_fields)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
			ON CONFLICT (connector_id, service, auth_type) DO UPDATE SET
				instructions_url = EXCLUDED.instructions_url,
				oauth_provider = EXCLUDED.oauth_provider,
				oauth_scopes = EXCLUDED.oauth_scopes,
				credential_fields = EXCLUDED.credential_fields`,
			m.ID, c.Service, c.AuthType, nilIfEmpty(c.InstructionsURL), nilIfEmpty(c.OAuthProvider), c.OAuthScopes, fieldsVal)
		if err != nil {
			return err
		}
	}

	// Remove credentials no longer in the manifest.
	// Build parallel arrays of service and auth_type for the WHERE clause.
	if len(credKeys) > 0 {
		services := make([]string, len(credKeys))
		authTypes := make([]string, len(credKeys))
		for i, k := range credKeys {
			services[i] = k.service
			authTypes[i] = k.authType
		}
		_, err = tx.Exec(ctx, `
			DELETE FROM connector_required_credentials
			WHERE connector_id = $1
			  AND NOT EXISTS (
			    SELECT 1
			    FROM unnest($2::text[], $3::text[]) AS keep(service, auth_type)
			    WHERE keep.service = connector_required_credentials.service
			      AND keep.auth_type = connector_required_credentials.auth_type
			  )`,
			m.ID, services, authTypes)
	} else {
		_, err = tx.Exec(ctx,
			`DELETE FROM connector_required_credentials WHERE connector_id = $1`, m.ID)
	}
	if err != nil {
		return err
	}

	// Upsert configuration templates.
	templateIDs := make([]string, 0, len(m.Templates))
	for _, tpl := range m.Templates {
		templateIDs = append(templateIDs, tpl.ID)

		// Guard against cross-connector ID collisions: if a template with
		// this ID already exists for a different connector, fail loudly
		// rather than silently reassigning it.
		var existingConnector *string
		_ = tx.QueryRow(ctx,
			`SELECT connector_id FROM action_config_templates WHERE id = $1`, tpl.ID,
		).Scan(&existingConnector)
		if existingConnector != nil && *existingConnector != m.ID {
			return fmt.Errorf("template id %q already belongs to connector %q, cannot reassign to %q", tpl.ID, *existingConnector, m.ID)
		}

		// Defensively default empty parameters to '{}' to honour the NOT NULL constraint.
		params := tpl.Parameters
		if len(params) == 0 {
			params = []byte("{}")
		}

		saSpec := nilIfEmptyBytes(tpl.StandingApprovalSpec)

		_, err := tx.Exec(ctx, `
			INSERT INTO action_config_templates (id, connector_id, action_type, name, description, parameters, standing_approval_spec)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE SET
				action_type = EXCLUDED.action_type,
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				parameters = EXCLUDED.parameters,
				standing_approval_spec = EXCLUDED.standing_approval_spec`,
			tpl.ID, m.ID, tpl.ActionType, tpl.Name, nilIfEmpty(tpl.Description), params, saSpec)
		if err != nil {
			return err
		}
	}

	// Remove templates no longer in the manifest.
	if len(templateIDs) > 0 {
		_, err = tx.Exec(ctx,
			`DELETE FROM action_config_templates WHERE connector_id = $1 AND id != ALL($2)`,
			m.ID, templateIDs)
	} else {
		_, err = tx.Exec(ctx,
			`DELETE FROM action_config_templates WHERE connector_id = $1`, m.ID)
	}
	if err != nil {
		return err
	}

	if owned {
		return CommitTx(ctx, tx)
	}
	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilIfEmptyBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}
	return b
}
