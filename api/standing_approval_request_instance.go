package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip/db"
)

type standingApprovalRequestInstanceResolution struct {
	ConnectorInstanceID      *string
	ConnectorInstanceDisplay *string
}

// resolveConnectorInstanceForStandingApprovalRequest resolves an explicit
// connector_instance selector on a rule proposal. Unlike applyConnectorInstanceToAction,
// this never applies default-instance fallback or "instance required" errors when
// the selector is omitted — callers keep the action-type-only display path.
func resolveConnectorInstanceForStandingApprovalRequest(
	ctx context.Context,
	d db.DBTX,
	agentID int64,
	userID, actionType, selector string,
) (*standingApprovalRequestInstanceResolution, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, nil
	}

	connectorIDPtr := connectorIDFromActionType(actionType)
	if connectorIDPtr == nil {
		return nil, &connectorInstanceResolutionError{
			status: http.StatusBadRequest,
			resp:   BadRequest(ErrInvalidRequest, "action_type does not identify a connector"),
		}
	}
	connectorID := *connectorIDPtr

	inst, err := db.ResolveAgentConnectorInstance(ctx, d, agentID, userID, connectorID, selector)
	if err != nil {
		if amb := ambiguousConnectorInstanceErr(err); amb != nil {
			return nil, amb
		}
		return nil, err
	}
	if inst == nil {
		return nil, &connectorInstanceResolutionError{
			status: http.StatusBadRequest,
			resp:   BadRequest(ErrConnectorInstanceNotFound, "no connector instance matches the given connector_instance"),
		}
	}

	display := strings.TrimSpace(inst.DisplayName)
	out := &standingApprovalRequestInstanceResolution{
		ConnectorInstanceID: &inst.ConnectorInstanceID,
	}
	if display != "" {
		out.ConnectorInstanceDisplay = &display
	}
	return out, nil
}

func respondConnectorInstanceResolutionError(w http.ResponseWriter, r *http.Request, err error) bool {
	var resErr *connectorInstanceResolutionError
	if errors.As(err, &resErr) {
		RespondError(w, r, resErr.status, resErr.resp)
		return true
	}
	return false
}
