package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

// preflightApprovalCredentials resolves and validates credentials that would be
// used when the approval is executed. Call before creating a pending approval
// or before committing an approve decision so users are not notified for
// actions that cannot run (e.g. OAuth needs_reauth).
func preflightApprovalCredentials(ctx context.Context, deps *Deps, agentID int64, userID, actionType, connectorInstanceID string) error {
	if deps == nil || deps.Connectors == nil {
		return nil
	}

	reqCreds, err := db.GetRequiredCredentialsByActionType(ctx, deps.DB, actionType)
	if err != nil {
		return fmt.Errorf("look up required credentials: %w", err)
	}
	if len(reqCreds) == 0 {
		return nil
	}

	connectorID, _, ok := strings.Cut(actionType, ".")
	if !ok {
		return &connectors.ValidationError{Message: "invalid action type"}
	}

	resolved, err := resolveCredentialsForExecution(ctx, deps, agentID, userID, actionType, connectorID, connectorInstanceID, reqCreds)
	if err != nil {
		return err
	}

	if conn, ok := deps.Connectors.Get(connectorID); ok {
		if err := conn.ValidateCredentials(ctx, resolved.Credentials); err != nil {
			return err
		}
	}
	return nil
}

func preflightApprovalFromStoredAction(ctx context.Context, deps *Deps, appr *db.Approval) error {
	actionType := actionTypeFromJSON(appr.Action)
	connectorInstanceID := connectorInstanceIDFromApprovalAction(appr.Action)
	return preflightApprovalCredentials(ctx, deps, appr.AgentID, appr.ApproverID, actionType, connectorInstanceID)
}

func connectorInstanceIDFromApprovalAction(action []byte) string {
	if len(action) == 0 {
		return ""
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(action, &raw) != nil {
		return ""
	}
	v, ok := raw["_connector_instance_id"]
	if !ok {
		return ""
	}
	var id string
	if json.Unmarshal(v, &id) != nil {
		return ""
	}
	return id
}

// respondApprovalCredentialPreflight maps credential preflight errors to HTTP
// responses for agent and dashboard approval endpoints. Returns true when a
// response was written.
func respondApprovalCredentialPreflight(w http.ResponseWriter, r *http.Request, actionType string, agentID int64, err error) bool {
	if err == nil {
		return false
	}

	cc := ConnectorContext{ActionType: actionType, AgentID: agentID}
	if handleConnectorError(w, r, err, cc) {
		return true
	}

	var ve *connectors.ValidationError
	if errors.As(err, &ve) {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, ve.Message))
		return true
	}

	log.Printf("[%s] approval credential preflight: %v", TraceID(r.Context()), err)
	CaptureError(r.Context(), err)
	RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to validate connector credentials"))
	return true
}
