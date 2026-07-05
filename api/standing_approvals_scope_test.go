package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func setupActionConfigScopeTest(t *testing.T) (db.DBTX, *Deps, http.Handler, int64, string, string, string, *db.AgentConnectorInstance, *db.AgentConnectorInstance) {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	actionType := connID + ".scope_action"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Scope Action")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	instOther, err := db.CreateAgentConnectorInstance(ctx, tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID,
	})
	if err != nil {
		t.Fatalf("CreateAgentConnectorInstance: %v", err)
	}

	instDefault, err := db.GetDefaultAgentConnectorInstance(ctx, tx, agentID, uid, connID)
	if err != nil || instDefault == nil {
		t.Fatalf("default instance: %v", instDefault)
	}

	deps := &Deps{DB: tx, JWTSigningSecret: testJWTSecret}
	return tx, deps, NewRouter(deps), agentID, uid, connID, actionType, instDefault, instOther
}

func TestUpdateStandingApproval_ScopeOmittedPreserves(t *testing.T) {
	t.Parallel()
	tx, deps, router, agentID, uid, connID, actionType, instDefault, instOther := setupActionConfigScopeTest(t)
	_ = deps

	configID := testhelper.GenerateID(t, "ac_")
	params, _ := json.Marshal(map[string]string{"connector_instance": instOther.ConnectorInstanceID})
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, actionType, testhelper.ActionConfigOpts{
		Parameters: params,
		Name:       "Scoped config",
	})

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:                  actionType,
		SourceActionConfigurationID: &configID,
		Constraints:                 []byte(`{"ping":"*"}`),
	})

	if _, err := tx.Exec(context.Background(),
		`UPDATE standing_approvals SET connector_instance_id = $1 WHERE standing_approval_id = $2`,
		instOther.ConnectorInstanceID, saID,
	); err != nil {
		t.Fatalf("seed connector_instance_id: %v", err)
	}

	body := `{"constraints":{"ping":"*"},"expires_at":null}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/update", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ConnectorInstanceID == nil || *resp.ConnectorInstanceID != instOther.ConnectorInstanceID {
		t.Fatalf("expected scope preserved, got %v", resp.ConnectorInstanceID)
	}
	_ = instDefault
}

func TestUpdateStandingApproval_ScopeNullSetsAllAccounts(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, uid, connID, actionType, _, instOther := setupActionConfigScopeTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:                  actionType,
		SourceActionConfigurationID: &configID,
		Constraints:                 []byte(`{"ping":"*"}`),
	})
	if _, err := tx.Exec(context.Background(),
		`UPDATE standing_approvals SET connector_instance_id = $1 WHERE standing_approval_id = $2`,
		instOther.ConnectorInstanceID, saID,
	); err != nil {
		t.Fatalf("seed connector_instance_id: %v", err)
	}

	body := `{"constraints":{"ping":"*"},"connector_instance_id":null}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/update", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ConnectorInstanceID != nil {
		t.Fatalf("expected all accounts (null scope), got %v", resp.ConnectorInstanceID)
	}
}

func TestUpdateStandingApproval_ScopeToSpecificInstance(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, uid, connID, actionType, _, instOther := setupActionConfigScopeTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, actionType)

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:                  actionType,
		SourceActionConfigurationID: &configID,
		Constraints:                 []byte(`{"ping":"*"}`),
	})

	body := `{"constraints":{"ping":"*"},"connector_instance_id":"` + instOther.ConnectorInstanceID + `"}`
	r := authenticatedJSONRequest(t, http.MethodPost, "/standing-approvals/"+saID+"/update", uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp standingApprovalResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ConnectorInstanceID == nil || *resp.ConnectorInstanceID != instOther.ConnectorInstanceID {
		t.Fatalf("expected scoped instance, got %v", resp.ConnectorInstanceID)
	}
}

func TestUpdateActionConfig_PropagatesScopeToLinkedStandingApproval(t *testing.T) {
	t.Parallel()
	tx, _, router, agentID, uid, connID, actionType, instDefault, instOther := setupActionConfigScopeTest(t)

	configID := testhelper.GenerateID(t, "ac_")
	params, _ := json.Marshal(map[string]string{"connector_instance": "*", "ping": "*"})
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, actionType, testhelper.ActionConfigOpts{
		Parameters: params,
		Name:       "All accounts config",
	})

	saID := testhelper.GenerateID(t, "sa_")
	testhelper.InsertStandingApprovalFull(t, tx, saID, agentID, uid, testhelper.StandingApprovalOpts{
		ActionType:                  actionType,
		SourceActionConfigurationID: &configID,
		Constraints:                 []byte(`{"ping":"*"}`),
	})

	newParams, _ := json.Marshal(map[string]string{
		"connector_instance": instOther.ConnectorInstanceID,
		"ping":               "*",
	})
	body := `{"parameters":` + string(newParams) + `}`
	r := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/"+configID, uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	sa, err := db.GetStandingApprovalByIDAndUser(context.Background(), tx, saID, uid)
	if err != nil {
		t.Fatalf("get standing approval: %v", err)
	}
	if sa == nil {
		t.Fatal("standing approval not found")
	}
	if sa.ConnectorInstanceID == nil || *sa.ConnectorInstanceID != instOther.ConnectorInstanceID {
		t.Fatalf("expected linked SA scope %q, got %v", instOther.ConnectorInstanceID, sa.ConnectorInstanceID)
	}

	// Narrow back to all accounts.
	allParams, _ := json.Marshal(map[string]string{"connector_instance": "*", "ping": "*"})
	body2 := `{"parameters":` + string(allParams) + `}`
	r2 := authenticatedJSONRequest(t, http.MethodPut, "/action-configurations/"+configID, uid, body2)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 on widen, got %d: %s", w2.Code, w2.Body.String())
	}

	sa2, err := db.GetStandingApprovalByIDAndUser(context.Background(), tx, saID, uid)
	if err != nil {
		t.Fatalf("get standing approval after widen: %v", err)
	}
	if sa2.ConnectorInstanceID != nil {
		t.Fatalf("expected linked SA all accounts, got %v", sa2.ConnectorInstanceID)
	}
	_ = instDefault
}

func TestConnectorInstanceIDPtrDifferent(t *testing.T) {
	t.Parallel()
	a := "11111111-1111-1111-1111-111111111111"
	b := "22222222-2222-2222-2222-222222222222"
	if !connectorInstanceIDPtrDifferent(nil, &a) {
		t.Error("nil vs id should differ")
	}
	if !connectorInstanceIDPtrDifferent(&a, &b) {
		t.Error("different ids should differ")
	}
	if connectorInstanceIDPtrDifferent(nil, nil) {
		t.Error("nil vs nil should not differ")
	}
	if connectorInstanceIDPtrDifferent(&a, &a) {
		t.Error("same id should not differ")
	}
}
