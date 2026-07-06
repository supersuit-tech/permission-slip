package api

import (
	"context"
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
	_ int64,
	_ string,
	actionType string,
) standingApprovalRequestDisplay {
	out := standingApprovalRequestDisplay{}

	schema, err := db.GetActionParametersSchema(ctx, d, actionType)
	if err != nil || schema == nil {
		connectorIDPtr := connectorIDFromActionType(actionType)
		if connectorIDPtr == nil {
			return out
		}
		schema = &db.ActionSchema{ConnectorID: *connectorIDPtr}
	}

	if conn, err := db.GetConnectorByID(ctx, d, schema.ConnectorID); err == nil && conn != nil {
		out.ConnectorName = strings.TrimSpace(conn.Name)
	}

	return out
}
