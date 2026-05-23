package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

// InsertStandingApprovalRequest inserts a pending standing approval request.
func InsertStandingApprovalRequest(t *testing.T, d db.DBTX, requestID string, agentID int64, userID, actionType string, constraints []byte) {
	t.Helper()
	_, err := db.InsertStandingApprovalRequest(t.Context(), d, db.InsertStandingApprovalRequestParams{
		RequestID:     requestID,
		AgentID:       agentID,
		UserID:        userID,
		ActionType:    actionType,
		ActionVersion: "1",
		Constraints:   constraints,
	})
	if err != nil {
		t.Fatalf("InsertStandingApprovalRequest: %v", err)
	}
}
