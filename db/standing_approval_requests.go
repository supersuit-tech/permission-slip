package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// StandingApprovalRequest represents a row from standing_approval_requests.
type StandingApprovalRequest struct {
	RequestID                   string
	AgentID                     int64
	UserID                      string
	ActionType                  string
	ActionVersion               string
	Constraints                 []byte
	SourceActionConfigurationID *string
	ConnectorName               *string
	ConnectorInstanceID         *string
	ConnectorInstanceDisplay    *string
	Status                      string
	DecidedAt                   *time.Time
	ResultingStandingApprovalID *string
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

const standingApprovalRequestColumns = `request_id, agent_id, user_id, action_type, action_version,
	constraints, source_action_configuration_id,
	connector_name, connector_instance_id, connector_instance_display,
	status, decided_at, resulting_standing_approval_id, created_at, updated_at`

// StandingApprovalRequestCursor identifies pagination position.
type StandingApprovalRequestCursor struct {
	CreatedAt time.Time
	RequestID string
}

// StandingApprovalRequestPage holds a page of requests.
type StandingApprovalRequestPage struct {
	Requests []StandingApprovalRequest
	HasMore  bool
}

const (
	StandingApprovalRequestStatusPending   = "pending"
	StandingApprovalRequestStatusApproved  = "approved"
	StandingApprovalRequestStatusDenied    = "denied"
	StandingApprovalRequestStatusExpired   = "expired"
	StandingApprovalRequestStatusCancelled = "cancelled"
)

// StandingApprovalRequestError is a domain error for request operations.
type StandingApprovalRequestError struct {
	Code   string
	Status string
}

func (e *StandingApprovalRequestError) Error() string { return e.Code }

const (
	StandingApprovalRequestErrNotFound        = "not_found"
	StandingApprovalRequestErrAlreadyResolved = "already_resolved"
	StandingApprovalRequestErrForbidden       = "forbidden"
)

func scanStandingApprovalRequest(row rowScanner) (*StandingApprovalRequest, error) {
	var sar StandingApprovalRequest
	var sourceConfigID, connectorName, connectorInstanceID, connectorInstanceDisplay, resultingSAID sql.NullString
	var decidedAt, createdAt, updatedAt sql.NullString
	err := row.Scan(
		&sar.RequestID, &sar.AgentID, &sar.UserID, &sar.ActionType, &sar.ActionVersion,
		&sar.Constraints, &sourceConfigID,
		&connectorName, &connectorInstanceID, &connectorInstanceDisplay,
		&sar.Status, &decidedAt, &resultingSAID, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if sourceConfigID.Valid {
		s := sourceConfigID.String
		sar.SourceActionConfigurationID = &s
	}
	if connectorName.Valid {
		s := connectorName.String
		sar.ConnectorName = &s
	}
	if connectorInstanceID.Valid {
		s := connectorInstanceID.String
		sar.ConnectorInstanceID = &s
	}
	if connectorInstanceDisplay.Valid {
		s := connectorInstanceDisplay.String
		sar.ConnectorInstanceDisplay = &s
	}
	if resultingSAID.Valid {
		s := resultingSAID.String
		sar.ResultingStandingApprovalID = &s
	}
	var err2 error
	sar.DecidedAt, err2 = sqliteTimePtr(decidedAt)
	if err2 != nil {
		return nil, err2
	}
	sar.CreatedAt, err2 = sqliteTimeRequired(createdAt)
	if err2 != nil {
		return nil, err2
	}
	sar.UpdatedAt, err2 = sqliteTimeRequired(updatedAt)
	if err2 != nil {
		return nil, err2
	}
	return &sar, nil
}

// InsertStandingApprovalRequestParams holds insert parameters.
type InsertStandingApprovalRequestParams struct {
	RequestID                   string
	AgentID                     int64
	UserID                      string
	ActionType                  string
	ActionVersion               string
	Constraints                 []byte
	SourceActionConfigurationID *string
	ConnectorName               *string
	ConnectorInstanceID         *string
	ConnectorInstanceDisplay    *string
}

// InsertStandingApprovalRequest inserts a pending standing approval request.
func InsertStandingApprovalRequest(ctx context.Context, db DBTX, p InsertStandingApprovalRequestParams) (*StandingApprovalRequest, error) {
	row := db.QueryRow(ctx,
		`INSERT INTO standing_approval_requests
		   (request_id, agent_id, user_id, action_type, action_version, constraints,
		    source_action_configuration_id,
		    connector_name, connector_instance_id, connector_instance_display, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending')
		 RETURNING `+standingApprovalRequestColumns,
		p.RequestID, p.AgentID, p.UserID, p.ActionType, p.ActionVersion, p.Constraints,
		p.SourceActionConfigurationID, p.ConnectorName, p.ConnectorInstanceID, p.ConnectorInstanceDisplay,
	)
	return scanStandingApprovalRequest(row)
}

// GetStandingApprovalRequestByIDAndUser returns a request for the given user or nil.
func GetStandingApprovalRequestByIDAndUser(ctx context.Context, db DBTX, requestID, userID string) (*StandingApprovalRequest, error) {
	row := db.QueryRow(ctx,
		`SELECT `+standingApprovalRequestColumns+`
		 FROM standing_approval_requests
		 WHERE request_id = $1 AND user_id = $2`,
		requestID, userID,
	)
	sar, err := scanStandingApprovalRequest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return sar, err
}

// ListStandingApprovalRequestsByUser returns paginated requests for a user.
func ListStandingApprovalRequestsByUser(ctx context.Context, db DBTX, userID, statusFilter string, limit int, cursor *StandingApprovalRequestCursor) (*StandingApprovalRequestPage, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	fetchLimit := limit + 1

	b := &queryBuilder{}
	b.addArg(userID)
	where := []string{"user_id = $1"}

	switch statusFilter {
	case "", "pending":
		where = append(where, "status = 'pending'")
	case "all":
	default:
		p := b.addArg(statusFilter)
		where = append(where, "status = "+p)
	}

	if cursor != nil {
		tsPlaceholder := b.addArg(TimestampForSQLite(cursor.CreatedAt))
		idPlaceholder := b.addArg(cursor.RequestID)
		where = append(where, fmt.Sprintf("(created_at, request_id) < (%s, %s)", tsPlaceholder, idPlaceholder))
	}

	limitPlaceholder := b.addArg(fetchLimit)
	query := fmt.Sprintf(
		`SELECT %s FROM standing_approval_requests WHERE %s ORDER BY created_at DESC, request_id DESC LIMIT %s`,
		standingApprovalRequestColumns, strings.Join(where, " AND "), limitPlaceholder,
	)

	rows, err := db.Query(ctx, query, b.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []StandingApprovalRequest
	for rows.Next() {
		sar, err := scanStandingApprovalRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, *sar)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hasMore := len(requests) > limit
	if hasMore {
		requests = requests[:limit]
	}
	return &StandingApprovalRequestPage{Requests: requests, HasMore: hasMore}, nil
}

// ApproveStandingApprovalRequestParams holds approve mutation parameters.
type ApproveStandingApprovalRequestParams struct {
	RequestID                   string
	UserID                      string
	StandingApprovalID          string
	SourceActionConfigurationID string
	ExpiresAt                   *time.Time
}

// ApproveStandingApprovalRequest marks a pending request approved and links the created standing approval.
// Returns the updated request. Idempotent: if already approved with same standing approval, returns success.
func ApproveStandingApprovalRequest(ctx context.Context, db DBTX, p ApproveStandingApprovalRequestParams) (*StandingApprovalRequest, error) {
	now := time.Now().UTC()
	row := db.QueryRow(ctx,
		`UPDATE standing_approval_requests
		 SET status = 'approved',
		     decided_at = $3,
		     resulting_standing_approval_id = $4,
		     updated_at = $3
		 WHERE request_id = $1 AND user_id = $2 AND status = 'pending'
		 RETURNING `+standingApprovalRequestColumns,
		p.RequestID, p.UserID, TimestampForSQLite(now), p.StandingApprovalID,
	)
	sar, err := scanStandingApprovalRequest(row)
	if err == nil {
		return sar, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Not pending — return existing row if already approved (idempotent).
	existing, err := GetStandingApprovalRequestByIDAndUser(ctx, db, p.RequestID, p.UserID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, &StandingApprovalRequestError{Code: StandingApprovalRequestErrNotFound}
	}
	if existing.Status == StandingApprovalRequestStatusApproved {
		return existing, nil
	}
	return nil, &StandingApprovalRequestError{Code: StandingApprovalRequestErrAlreadyResolved, Status: existing.Status}
}

// DenyStandingApprovalRequest marks a pending request denied.
func DenyStandingApprovalRequest(ctx context.Context, db DBTX, requestID, userID string) (*StandingApprovalRequest, error) {
	now := time.Now().UTC()
	row := db.QueryRow(ctx,
		`UPDATE standing_approval_requests
		 SET status = 'denied', decided_at = $3, updated_at = $3
		 WHERE request_id = $1 AND user_id = $2 AND status = 'pending'
		 RETURNING `+standingApprovalRequestColumns,
		requestID, userID, TimestampForSQLite(now),
	)
	sar, err := scanStandingApprovalRequest(row)
	if err == nil {
		return sar, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	existing, err := GetStandingApprovalRequestByIDAndUser(ctx, db, requestID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, &StandingApprovalRequestError{Code: StandingApprovalRequestErrNotFound}
	}
	if existing.Status == StandingApprovalRequestStatusDenied {
		return existing, nil
	}
	return nil, &StandingApprovalRequestError{Code: StandingApprovalRequestErrAlreadyResolved, Status: existing.Status}
}

// FindActionConfigIDForStandingApprovalRequest resolves source_action_configuration_id
// for approving a request. Uses explicit ID when set; otherwise finds a unique config
// for the agent and action type owned by the user.
func FindActionConfigIDForStandingApprovalRequest(ctx context.Context, db DBTX, agentID int64, userID, actionType string, explicitID *string) (string, error) {
	if explicitID != nil && *explicitID != "" {
		ac, err := GetActionConfigByID(ctx, db, *explicitID, userID)
		if err != nil {
			return "", err
		}
		if ac == nil {
			return "", fmt.Errorf("source_action_configuration_id not found")
		}
		if ac.AgentID != agentID {
			return "", fmt.Errorf("action configuration does not belong to agent")
		}
		if ac.ActionType != actionType {
			return "", fmt.Errorf("action configuration action_type mismatch")
		}
		return *explicitID, nil
	}

	configs, err := ListActionConfigsByAgent(ctx, db, agentID, userID)
	if err != nil {
		return "", err
	}
	var matches []string
	for _, ac := range configs {
		if ac.ActionType == actionType {
			matches = append(matches, ac.ID)
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no action configuration found for action_type; create one in the dashboard or pass source_action_configuration_id")
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple action configurations match action_type; pass source_action_configuration_id")
	}
}
