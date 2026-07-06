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

	connectorIDPtr := connectorIDFromActionType(actionType)
	if connectorIDPtr == nil {
		return out
	}

	if conn, err := db.GetConnectorByID(ctx, d, *connectorIDPtr); err == nil && conn != nil {
		out.ConnectorName = strings.TrimSpace(conn.Name)
	}

	return out
}
