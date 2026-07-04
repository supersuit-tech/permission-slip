package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/supersuit-tech/permission-slip/db"
)

type standingApprovalRequestDisplay struct {
	ConnectorName            string
	ConnectorInstanceDisplay string
}

func resolveStandingApprovalRequestDisplay(
	ctx context.Context,
	d db.DBTX,
	agentID int64,
	userID, actionType string,
	sourceConfigID *string,
) standingApprovalRequestDisplay {
	out := standingApprovalRequestDisplay{}

	connectorIDPtr := connectorIDFromActionType(actionType)
	if connectorIDPtr == nil && sourceConfigID == nil {
		return out
	}

	if sourceConfigID != nil {
		configID := strings.TrimSpace(*sourceConfigID)
		if configID != "" {
			ac, err := db.GetActionConfigByID(ctx, d, configID, userID)
			if err == nil && ac != nil && ac.AgentID == agentID {
				if conn, err := db.GetConnectorByID(ctx, d, ac.ConnectorID); err == nil && conn != nil {
					out.ConnectorName = strings.TrimSpace(conn.Name)
				}
				out.ConnectorInstanceDisplay = resolveConnectorInstanceDisplayFromActionConfig(
					ctx, d, agentID, userID, ac,
				)
				return out
			}
		}
	}

	if connectorIDPtr != nil {
		if conn, err := db.GetConnectorByID(ctx, d, *connectorIDPtr); err == nil && conn != nil {
			out.ConnectorName = strings.TrimSpace(conn.Name)
		}
	}

	return out
}

func resolveStandingApprovalRequestDisplayLegacy(
	ctx context.Context,
	d db.DBTX,
	agentID int64,
	userID, actionType string,
	sourceConfigID *string,
) standingApprovalRequestDisplay {
	out := standingApprovalRequestDisplay{}

	connectorIDPtr := connectorIDFromActionType(actionType)
	if connectorIDPtr == nil {
		return out
	}
	connectorID := *connectorIDPtr

	if conn, err := db.GetConnectorByID(ctx, d, connectorID); err == nil && conn != nil {
		out.ConnectorName = strings.TrimSpace(conn.Name)
	}

	if sourceConfigID == nil {
		return out
	}
	configID := strings.TrimSpace(*sourceConfigID)
	if configID == "" {
		return out
	}

	ac, err := db.GetActionConfigByID(ctx, d, configID, userID)
	if err != nil || ac == nil || ac.AgentID != agentID {
		return out
	}

	out.ConnectorInstanceDisplay = resolveConnectorInstanceDisplayFromActionConfig(
		ctx, d, agentID, userID, ac,
	)
	return out
}

func resolveConnectorInstanceDisplayFromActionConfig(
	ctx context.Context,
	d db.DBTX,
	agentID int64,
	userID string,
	ac *db.ActionConfiguration,
) string {
	if ac == nil {
		return ""
	}

	instances, err := db.ListAgentConnectorInstances(ctx, d, agentID, userID, ac.ConnectorID)
	if err != nil || len(instances) == 0 {
		return ""
	}

	selector := fixedConnectorInstanceSelector(ac.Parameters)
	if selector != "" {
		inst, err := db.ResolveAgentConnectorInstance(ctx, d, agentID, userID, ac.ConnectorID, selector)
		if err != nil || inst == nil {
			return ""
		}
		return strings.TrimSpace(inst.DisplayName)
	}

	return ""
}

func fixedConnectorInstanceSelector(parameters []byte) string {
	if len(parameters) == 0 {
		return ""
	}
	var params map[string]json.RawMessage
	if err := json.Unmarshal(parameters, &params); err != nil {
		return ""
	}
	raw, ok := params["connector_instance"]
	if !ok || db.IsWildcard(raw) || db.IsPattern(raw) {
		return ""
	}
	var selector string
	if err := json.Unmarshal(raw, &selector); err != nil {
		return ""
	}
	return strings.TrimSpace(selector)
}
