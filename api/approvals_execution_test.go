package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// blockingMockAction waits until the execution context expires, simulating a
// connector that hangs until the execution budget is exhausted (e.g. iMessage
// blocked on a macOS Automation prompt).
type blockingMockAction struct {
	called bool
}

func (a *blockingMockAction) Execute(ctx context.Context, _ connectors.ActionRequest) (*connectors.ActionResult, error) {
	a.called = true
	<-ctx.Done()
	return nil, &connectors.TimeoutError{Message: "connector blocked until execution deadline exceeded"}
}

func insertPendingApprovalWithActionType(t *testing.T, d db.DBTX, approvalID string, agentID int64, approverID, actionType string) {
	t.Helper()
	action := fmt.Sprintf(`{"type":%q,"version":"1","parameters":{}}`, actionType)
	testhelper.MustExec(t, d,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at)
		 VALUES ($1, $2, $3, $4, '{"description":"test"}', 'pending', strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+1 hour'))`,
		approvalID, agentID, approverID, action)
}

func TestExecuteApprovalAction_PersistsResultAfterExecutionTimeout(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	apprID := testhelper.GenerateID(t, "appr_")
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	const actionType = "testconn.block"
	insertPendingApprovalWithActionType(t, tx, apprID, agentID, uid, actionType)

	testhelper.InsertConnector(t, tx, "testconn")
	testhelper.InsertConnectorAction(t, tx, "testconn", actionType, "Block")

	action := &blockingMockAction{}
	registry := connectors.NewRegistry()
	registry.Register(&mockConnector{
		id:      "testconn",
		actions: map[string]connectors.Action{actionType: action},
	})

	deps := &Deps{DB: tx, Connectors: registry, JWTSigningSecret: testJWTSecret}

	appr, _, err := db.ApproveApproval(t.Context(), tx, apprID, uid)
	if err != nil {
		t.Fatalf("ApproveApproval: %v", err)
	}

	// Use a short deadline to keep the test fast; same race as the 30s production budget.
	execCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	execStatus, execResultJSON := executeApprovalAction(execCtx, deps, uid, appr)

	if execStatus != "error" {
		t.Fatalf("expected execution status error, got %q", execStatus)
	}
	if !action.called {
		t.Fatal("expected connector action to be invoked")
	}

	var result map[string]string
	if err := json.Unmarshal(execResultJSON, &result); err != nil {
		t.Fatalf("unmarshal execution result: %v", err)
	}
	if !strings.Contains(result["error"], "blocked until execution deadline") {
		t.Fatalf("expected timeout message in execution result, got %q", result["error"])
	}

	stored, err := db.GetApprovalByIDAndApprover(context.Background(), tx, apprID, uid)
	if err != nil {
		t.Fatalf("GetApprovalByIDAndApprover: %v", err)
	}
	if stored == nil {
		t.Fatal("expected approval to exist")
	}
	if stored.ExecutionStatus == nil || *stored.ExecutionStatus != "error" {
		t.Fatalf("expected persisted execution_status error, got %+v", stored.ExecutionStatus)
	}
	if stored.ExecutedAt == nil {
		t.Fatal("expected executed_at to be set after persistence")
	}
	if len(stored.ExecutionResult) == 0 {
		t.Fatal("expected execution_result to be persisted")
	}
	var storedResult map[string]string
	if err := json.Unmarshal(stored.ExecutionResult, &storedResult); err != nil {
		t.Fatalf("unmarshal stored execution result: %v", err)
	}
	if !strings.Contains(storedResult["error"], "blocked until execution deadline") {
		t.Fatalf("expected timeout message in stored execution_result, got %q", storedResult["error"])
	}
}
