package api

import (
	"context"
	"net/http"

	"github.com/supersuit-tech/permission-slip/db"
)

// resolveConnectorInstanceIDFromConfigParameters returns the connector_instance_id
// for a fixed connector_instance selector in config parameters, or nil for all accounts.
func resolveConnectorInstanceIDFromConfigParameters(
	ctx context.Context,
	d db.DBTX,
	agentID int64,
	userID, connectorID string,
	parameters []byte,
) (*string, error) {
	selector := fixedConnectorInstanceSelector(parameters)
	if selector == "" {
		return nil, nil
	}

	inst, err := db.ResolveAgentConnectorInstance(ctx, d, agentID, userID, connectorID, selector)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, &connectorInstanceResolutionError{
			status: http.StatusBadRequest,
			resp:   BadRequest(ErrConnectorInstanceNotFound, "no connector instance matches the given connector_instance"),
		}
	}
	id := inst.ConnectorInstanceID
	return &id, nil
}

// connectorInstanceScopeChanged reports whether the account scope implied by
// parameters.connector_instance differs between old and new parameter blobs.
func connectorInstanceScopeChanged(
	ctx context.Context,
	d db.DBTX,
	agentID int64,
	userID, connectorID string,
	oldParams, newParams []byte,
) (bool, error) {
	oldID, err := resolveConnectorInstanceIDFromConfigParameters(ctx, d, agentID, userID, connectorID, oldParams)
	if err != nil {
		return false, err
	}
	newID, err := resolveConnectorInstanceIDFromConfigParameters(ctx, d, agentID, userID, connectorID, newParams)
	if err != nil {
		return false, err
	}
	return connectorInstanceIDPtrDifferent(oldID, newID), nil
}

func connectorInstanceIDPtrDifferent(a, b *string) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}

// validateStandingApprovalConnectorInstanceID ensures the instance belongs to the
// standing approval's agent and connector when setting a non-null scope.
func validateStandingApprovalConnectorInstanceID(
	ctx context.Context,
	d db.DBTX,
	sa *db.StandingApproval,
	userID string,
	connectorInstanceID *string,
) error {
	if connectorInstanceID == nil {
		return nil
	}

	connectorIDPtr := connectorIDFromActionType(sa.ActionType)
	if connectorIDPtr == nil {
		return &connectorInstanceResolutionError{
			status: http.StatusBadRequest,
			resp:   BadRequest(ErrInvalidRequest, "action_type does not identify a connector"),
		}
	}

	inst, err := db.GetAgentConnectorInstance(ctx, d, sa.AgentID, userID, *connectorIDPtr, *connectorInstanceID)
	if err != nil {
		return err
	}
	if inst == nil {
		return &connectorInstanceResolutionError{
			status: http.StatusBadRequest,
			resp:   BadRequest(ErrConnectorInstanceNotFound, "connector instance not found for this agent"),
		}
	}
	return nil
}
