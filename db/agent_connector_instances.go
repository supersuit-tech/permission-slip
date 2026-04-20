package db

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrAgentConnectorInstanceLabelRequired is returned when label is empty after trim.
var ErrAgentConnectorInstanceLabelRequired = errors.New("label is required")

// ErrAgentConnectorInstanceLabelTooLong is returned when label exceeds MaxAgentConnectorInstanceLabelLen runes.
var ErrAgentConnectorInstanceLabelTooLong = errors.New("label exceeds maximum length")

// AgentConnectorInstance is a row in agent_connectors (one instance of a connector type for an agent).
type AgentConnectorInstance struct {
	ConnectorInstanceID string
	AgentID             int64
	ConnectorID         string
	ApproverID          string
	Label               string
	IsDefault           bool
	EnabledAt           time.Time
}

// AgentConnectorInstanceError is a domain error for connector instance operations.
type AgentConnectorInstanceError struct {
	Code AgentConnectorInstanceErrCode
}

func (e *AgentConnectorInstanceError) Error() string { return string(e.Code) }

// AgentConnectorInstanceErrCode enumerates connector-instance-specific error codes.
type AgentConnectorInstanceErrCode string

const (
	AgentConnectorInstanceErrNotFound            AgentConnectorInstanceErrCode = "connector_instance_not_found"
	AgentConnectorInstanceErrDuplicateLabel      AgentConnectorInstanceErrCode = "duplicate_instance_label"
	AgentConnectorInstanceErrCannotDeleteDefault AgentConnectorInstanceErrCode = "cannot_delete_default_instance"
)

// MaxAgentConnectorInstanceLabelLen is the maximum rune length for instance labels (API + DB).
const MaxAgentConnectorInstanceLabelLen = 256

// CreateAgentConnectorInstanceParams holds parameters for creating a new instance (non-default row).
type CreateAgentConnectorInstanceParams struct {
	AgentID     int64
	ApproverID  string
	ConnectorID string
	Label       string
}

// CreateAgentConnectorInstance inserts a new agent connector instance with the given label.
// The first instance for an (agent, connector) pair is the default (enforced by DB trigger);
// additional instances are non-default.
func CreateAgentConnectorInstance(ctx context.Context, db DBTX, p CreateAgentConnectorInstanceParams) (*AgentConnectorInstance, error) {
	label := strings.TrimSpace(p.Label)
	if label == "" {
		return nil, ErrAgentConnectorInstanceLabelRequired
	}
	if utf8.RuneCountInString(label) > MaxAgentConnectorInstanceLabelLen {
		return nil, ErrAgentConnectorInstanceLabelTooLong
	}

	var agentOK bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM agents WHERE agent_id = $1 AND approver_id = $2)`,
		p.AgentID, p.ApproverID,
	).Scan(&agentOK); err != nil {
		return nil, err
	}
	if !agentOK {
		return nil, &AgentConnectorError{Code: AgentConnectorErrAgentNotFound}
	}

	var connOK bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM connectors WHERE id = $1)`,
		p.ConnectorID,
	).Scan(&connOK); err != nil {
		return nil, err
	}
	if !connOK {
		return nil, &AgentConnectorError{Code: AgentConnectorErrConnectorNotFound}
	}

	var enabled bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM agent_connectors
			WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3
		)`,
		p.AgentID, p.ApproverID, p.ConnectorID,
	).Scan(&enabled); err != nil {
		return nil, err
	}
	if !enabled {
		return nil, &AgentConnectorError{Code: AgentConnectorErrConnectorNotEnabled}
	}

	row := db.QueryRow(ctx, `
		INSERT INTO agent_connectors (agent_id, approver_id, connector_id, label, is_default)
		VALUES ($1, $2, $3, $4, false)
		RETURNING connector_instance_id, agent_id, connector_id, approver_id, label, is_default, enabled_at`,
		p.AgentID, p.ApproverID, p.ConnectorID, label,
	)
	inst, err := scanAgentConnectorInstance(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeUniqueViolation {
			return nil, &AgentConnectorInstanceError{Code: AgentConnectorInstanceErrDuplicateLabel}
		}
		return nil, err
	}
	return inst, nil
}

func scanAgentConnectorInstance(row pgx.Row) (*AgentConnectorInstance, error) {
	var inst AgentConnectorInstance
	err := row.Scan(
		&inst.ConnectorInstanceID, &inst.AgentID, &inst.ConnectorID, &inst.ApproverID,
		&inst.Label, &inst.IsDefault, &inst.EnabledAt,
	)
	if err != nil {
		return nil, err
	}
	return &inst, nil
}

// ListAgentConnectorInstances returns all instances for an agent+connector scoped to the approver.
func ListAgentConnectorInstances(ctx context.Context, db DBTX, agentID int64, approverID, connectorID string) ([]AgentConnectorInstance, error) {
	rows, err := db.Query(ctx, `
		SELECT connector_instance_id, agent_id, connector_id, approver_id, label, is_default, enabled_at
		FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3
		ORDER BY enabled_at ASC, connector_instance_id ASC`,
		agentID, approverID, connectorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AgentConnectorInstance
	for rows.Next() {
		var inst AgentConnectorInstance
		if err := rows.Scan(
			&inst.ConnectorInstanceID, &inst.AgentID, &inst.ConnectorID, &inst.ApproverID,
			&inst.Label, &inst.IsDefault, &inst.EnabledAt,
		); err != nil {
			return nil, err
		}
		out = append(out, inst)
	}
	return out, rows.Err()
}

// GetAgentConnectorInstance returns a single instance by ID, scoped to agent and approver.
func GetAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, connectorInstanceID string) (*AgentConnectorInstance, error) {
	row := db.QueryRow(ctx, `
		SELECT connector_instance_id, agent_id, connector_id, approver_id, label, is_default, enabled_at
		FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND connector_instance_id = $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	)
	inst, err := scanAgentConnectorInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// GetDefaultAgentConnectorInstance returns the default instance for an (agent, connector) pair.
func GetDefaultAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID string) (*AgentConnectorInstance, error) {
	row := db.QueryRow(ctx, `
		SELECT connector_instance_id, agent_id, connector_id, approver_id, label, is_default, enabled_at
		FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND is_default`,
		agentID, approverID, connectorID,
	)
	inst, err := scanAgentConnectorInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// GetDefaultAgentConnectorInstanceByAgent returns the default instance using only agent_id and connector_id.
// Each agent has a single approver; this is used when approver_id is not available on the call path.
func GetDefaultAgentConnectorInstanceByAgent(ctx context.Context, db DBTX, agentID int64, connectorID string) (*AgentConnectorInstance, error) {
	row := db.QueryRow(ctx, `
		SELECT connector_instance_id, agent_id, connector_id, approver_id, label, is_default, enabled_at
		FROM agent_connectors
		WHERE agent_id = $1 AND connector_id = $2 AND is_default`,
		agentID, connectorID,
	)
	inst, err := scanAgentConnectorInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// RenameAgentConnectorInstance updates the label for an instance.
func RenameAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, connectorInstanceID, newLabel string) (*AgentConnectorInstance, error) {
	label := strings.TrimSpace(newLabel)
	if label == "" {
		return nil, ErrAgentConnectorInstanceLabelRequired
	}
	if utf8.RuneCountInString(label) > MaxAgentConnectorInstanceLabelLen {
		return nil, ErrAgentConnectorInstanceLabelTooLong
	}

	row := db.QueryRow(ctx, `
		UPDATE agent_connectors
		SET label = $5
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND connector_instance_id = $4::uuid
		RETURNING connector_instance_id, agent_id, connector_id, approver_id, label, is_default, enabled_at`,
		agentID, approverID, connectorID, connectorInstanceID, label,
	)
	inst, err := scanAgentConnectorInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeUniqueViolation {
			return nil, &AgentConnectorInstanceError{Code: AgentConnectorInstanceErrDuplicateLabel}
		}
		return nil, err
	}
	return inst, nil
}

// DeleteAgentConnectorInstance removes a non-default instance and revokes instance-scoped standing approvals.
func DeleteAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, connectorInstanceID string) error {
	tx, owned, err := BeginOrContinue(ctx, db)
	if err != nil {
		return err
	}
	if owned {
		defer RollbackTx(ctx, tx) //nolint:errcheck // commit path clears
	}

	var isDefault bool
	err = tx.QueryRow(ctx, `
		SELECT is_default FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND connector_instance_id = $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	).Scan(&isDefault)
	if errors.Is(err, pgx.ErrNoRows) {
		return &AgentConnectorInstanceError{Code: AgentConnectorInstanceErrNotFound}
	}
	if err != nil {
		return err
	}
	if isDefault {
		return &AgentConnectorInstanceError{Code: AgentConnectorInstanceErrCannotDeleteDefault}
	}

	_, err = tx.Exec(ctx, `
		UPDATE standing_approvals
		SET status = 'revoked', revoked_at = now()
		WHERE agent_id = $1
		  AND user_id = $2
		  AND status = 'active'
		  AND connector_instance_id = $3::uuid`,
		agentID, approverID, connectorInstanceID,
	)
	if err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND connector_instance_id = $4::uuid`,
		agentID, approverID, connectorID, connectorInstanceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return &AgentConnectorInstanceError{Code: AgentConnectorInstanceErrNotFound}
	}

	if owned {
		return CommitTx(ctx, tx)
	}
	return nil
}

// ResolveAgentConnectorInstance finds an instance by UUID or by exact label (trimmed).
func ResolveAgentConnectorInstance(ctx context.Context, db DBTX, agentID int64, approverID, connectorID, selector string) (*AgentConnectorInstance, error) {
	sel := strings.TrimSpace(selector)
	if sel == "" {
		return nil, nil
	}

	if _, err := uuid.Parse(sel); err == nil {
		return GetAgentConnectorInstance(ctx, db, agentID, approverID, connectorID, sel)
	}

	row := db.QueryRow(ctx, `
		SELECT connector_instance_id, agent_id, connector_id, approver_id, label, is_default, enabled_at
		FROM agent_connectors
		WHERE agent_id = $1 AND approver_id = $2 AND connector_id = $3 AND label = $4`,
		agentID, approverID, connectorID, sel,
	)
	inst, err := scanAgentConnectorInstance(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return inst, nil
}
