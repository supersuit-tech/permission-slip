package db_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

// ── Schema Tests ─────────────────────────────────────────────────────────────

func TestActionConfigurationsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "action_configurations", []string{
		"id", "agent_id", "user_id", "connector_id", "action_type",
		"parameters", "status", "name", "description",
		"created_at", "updated_at",
	})
}

func TestActionConfigurationsIndexAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "action_configurations", "idx_action_configurations_agent")
}

func TestActionConfigurationsIndexConnectorAction(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "action_configurations", "idx_action_configurations_connector_action")
}

// ── Status CHECK Constraint ──────────────────────────────────────────────────

func TestActionConfigurationsStatusCheckConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	base := testhelper.GenerateID(t, "ac_")
	testhelper.RequireCheckValues(t, tx, "status",
		[]string{"active", "disabled"}, "invalid",
		func(value string, i int) error {
			_, err := tx.Exec(context.Background(),
				`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, status, name)
				 VALUES ($1, $2, $3, $4, 'test.action', '{}', $5, 'test')`,
				fmt.Sprintf("%s_%d", base, i), agentID, uid, connID, value)
			return err
		})
}

// ── Cascade Delete Tests ─────────────────────────────────────────────────────

func TestActionConfigurationsCascadeDeleteOnAgentDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")
	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, "test.action")

	testhelper.RequireCascadeDeletes(t, tx,
		fmt.Sprintf("DELETE FROM agents WHERE agent_id = %d", agentID),
		[]string{"action_configurations"},
		fmt.Sprintf("agent_id = %d", agentID),
	)
}

func TestActionConfigurationsCascadeDeleteOnProfileDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")
	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, "test.action")

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM profiles WHERE id = '"+uid+"'",
		[]string{"action_configurations"},
		"user_id = '"+uid+"'",
	)
}

func TestActionConfigurationsCascadeDeleteOnConnectorDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")
	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, "test.action")

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM connectors WHERE id = '"+connID+"'",
		[]string{"action_configurations"},
		"connector_id = '"+connID+"'",
	)
}


// ── JSONB Size Constraint ────────────────────────────────────────────────────

func TestActionConfigurationsParametersSizeLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	// Build a large JSON object that exceeds 65536 bytes
	largeValue := make([]byte, 70000)
	for i := range largeValue {
		largeValue[i] = 'a'
	}
	largeJSON := fmt.Sprintf(`{"key": "%s"}`, string(largeValue))

	err := testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(context.Background(),
			`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, name)
			 VALUES ($1, $2, $3, $4, 'test.action', $5, 'test')`,
			testhelper.GenerateID(t, "ac_"), agentID, uid, connID, largeJSON)
		return err
	})
	if err == nil {
		t.Error("expected CHECK constraint violation for oversized parameters, but insert succeeded")
	}
}

// ── CreateActionConfig ───────────────────────────────────────────────────────

func TestCreateActionConfig_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	desc := "Test config description"
	ac, err := db.CreateActionConfig(t.Context(), tx, db.CreateActionConfigParams{
		ID:          configID,
		AgentID:     agentID,
		UserID:      uid,
		ConnectorID: connID,
		ActionType:  "test.action",
		Parameters:  []byte(`{"repo": "supersuit-tech/webapp", "label": "*"}`),
		Name:        "Test Config",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("CreateActionConfig: %v", err)
	}
	if ac.ID != configID {
		t.Errorf("expected id %q, got %q", configID, ac.ID)
	}
	if ac.AgentID != agentID {
		t.Errorf("expected agent_id %d, got %d", agentID, ac.AgentID)
	}
	if ac.UserID != uid {
		t.Errorf("expected user_id %q, got %q", uid, ac.UserID)
	}
	if ac.ConnectorID != connID {
		t.Errorf("expected connector_id %q, got %q", connID, ac.ConnectorID)
	}
	if ac.ActionType != "test.action" {
		t.Errorf("expected action_type 'test.action', got %q", ac.ActionType)
	}
	if ac.Status != "active" {
		t.Errorf("expected status 'active', got %q", ac.Status)
	}
	if ac.Name != "Test Config" {
		t.Errorf("expected name 'Test Config', got %q", ac.Name)
	}
	if ac.Description == nil || *ac.Description != desc {
		t.Errorf("expected description %q, got %v", desc, ac.Description)
	}
	if ac.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if ac.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateActionConfig_AgentNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	_, err := db.CreateActionConfig(t.Context(), tx, db.CreateActionConfigParams{
		ID:          testhelper.GenerateID(t, "ac_"),
		AgentID:     999999,
		UserID:      uid,
		ConnectorID: connID,
		ActionType:  "test.action",
		Parameters:  []byte(`{}`),
		Name:        "Should Fail",
	})
	if err == nil {
		t.Fatal("expected error for non-existent agent")
	}
	acErr, ok := err.(*db.ActionConfigError)
	if !ok {
		t.Fatalf("expected *ActionConfigError, got %T: %v", err, err)
	}
	if acErr.Code != db.ActionConfigErrAgentNotFound {
		t.Errorf("expected ActionConfigErrAgentNotFound, got %v", acErr.Code)
	}
}

func TestCreateActionConfig_WrongOwner(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	// User 2 tries to create a config for User 1's agent
	_, err := db.CreateActionConfig(t.Context(), tx, db.CreateActionConfigParams{
		ID:          testhelper.GenerateID(t, "ac_"),
		AgentID:     agentID,
		UserID:      uid2,
		ConnectorID: connID,
		ActionType:  "test.action",
		Parameters:  []byte(`{}`),
		Name:        "Wrong Owner",
	})
	if err == nil {
		t.Fatal("expected error for wrong owner")
	}
	acErr, ok := err.(*db.ActionConfigError)
	if !ok {
		t.Fatalf("expected *ActionConfigError, got %T: %v", err, err)
	}
	if acErr.Code != db.ActionConfigErrAgentNotFound {
		t.Errorf("expected ActionConfigErrAgentNotFound, got %v", acErr.Code)
	}
}

func TestCreateActionConfig_InvalidConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	err := testhelper.WithSavepoint(t, tx, func() error {
		_, err := db.CreateActionConfig(t.Context(), tx, db.CreateActionConfigParams{
			ID:          testhelper.GenerateID(t, "ac_"),
			AgentID:     agentID,
			UserID:      uid,
			ConnectorID: "nonexistent",
			ActionType:  "test.action",
			Parameters:  []byte(`{}`),
			Name:        "Bad Connector",
		})
		return err
	})
	if err == nil {
		t.Fatal("expected error for invalid connector")
	}
	acErr, ok := err.(*db.ActionConfigError)
	if !ok {
		t.Fatalf("expected *ActionConfigError, got %T: %v", err, err)
	}
	if acErr.Code != db.ActionConfigErrInvalidRef {
		t.Errorf("expected ActionConfigErrInvalidRef, got %v", acErr.Code)
	}
}

// ── GetActionConfigByID ──────────────────────────────────────────────────────

func TestGetActionConfigByID_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, "test.action")

	ac, err := db.GetActionConfigByID(t.Context(), tx, configID, uid)
	if err != nil {
		t.Fatalf("GetActionConfigByID: %v", err)
	}
	if ac == nil {
		t.Fatal("expected non-nil result")
	}
	if ac.ID != configID {
		t.Errorf("expected id %q, got %q", configID, ac.ID)
	}
	if ac.Status != "active" {
		t.Errorf("expected status 'active', got %q", ac.Status)
	}
}

func TestGetActionConfigByID_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ac, err := db.GetActionConfigByID(t.Context(), tx, "nonexistent", uid)
	if err != nil {
		t.Fatalf("GetActionConfigByID: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for non-existent config")
	}
}

func TestGetActionConfigByID_ScopedToUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid1, connID, "test.action")

	// User 2 should not see User 1's config
	ac, err := db.GetActionConfigByID(t.Context(), tx, configID, uid2)
	if err != nil {
		t.Fatalf("GetActionConfigByID: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for config belonging to different user")
	}
}

// ── ListActionConfigsByAgent ─────────────────────────────────────────────────

func TestListActionConfigsByAgent_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	configs, err := db.ListActionConfigsByAgent(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("ListActionConfigsByAgent: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(configs))
	}
}

func TestListActionConfigsByAgent_ReturnsConfigs(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action1", "Action 1")
	testhelper.InsertConnectorAction(t, tx, connID, "test.action2", "Action 2")

	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, "test.action1")
	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, "test.action2")

	configs, err := db.ListActionConfigsByAgent(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("ListActionConfigsByAgent: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(configs))
	}
}

func TestListActionConfigsByAgent_ScopedToUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agentID1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	agentID2 := testhelper.InsertUserWithAgent(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID1, uid1, connID, "test.action")
	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID2, uid2, connID, "test.action")

	// User 1 should only see their own agent's configs
	configs, err := db.ListActionConfigsByAgent(t.Context(), tx, agentID1, uid1)
	if err != nil {
		t.Fatalf("ListActionConfigsByAgent: %v", err)
	}
	if len(configs) != 1 {
		t.Errorf("expected 1 config for user1, got %d", len(configs))
	}

	// User 2 should not see User 1's agent configs
	configs, err = db.ListActionConfigsByAgent(t.Context(), tx, agentID1, uid2)
	if err != nil {
		t.Fatalf("ListActionConfigsByAgent: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs for user2 on user1's agent, got %d", len(configs))
	}
}

// ── UpdateActionConfig ───────────────────────────────────────────────────────

func TestUpdateActionConfig_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, "test.action")

	newName := "Updated Name"
	newStatus := "disabled"
	newDesc := "Updated description"
	ac, err := db.UpdateActionConfig(t.Context(), tx, db.UpdateActionConfigParams{
		ID:          configID,
		UserID:      uid,
		Parameters:  []byte(`{"repo": "updated/repo"}`),
		Status:      &newStatus,
		Name:        &newName,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("UpdateActionConfig: %v", err)
	}
	if ac.Name != newName {
		t.Errorf("expected name %q, got %q", newName, ac.Name)
	}
	if ac.Status != newStatus {
		t.Errorf("expected status %q, got %q", newStatus, ac.Status)
	}
	if ac.Description == nil || *ac.Description != newDesc {
		t.Errorf("expected description %q, got %v", newDesc, ac.Description)
	}
	if string(ac.Parameters) != `{"repo": "updated/repo"}` {
		t.Errorf("expected updated parameters, got %s", ac.Parameters)
	}
	if ac.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestUpdateActionConfig_PartialUpdate(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	desc := "Original description"
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, "test.action", testhelper.ActionConfigOpts{
		Name:        "Original Name",
		Description: &desc,
	})

	// Update only the name, leave everything else untouched
	newName := "Updated Name Only"
	ac, err := db.UpdateActionConfig(t.Context(), tx, db.UpdateActionConfigParams{
		ID:     configID,
		UserID: uid,
		Name:   &newName,
	})
	if err != nil {
		t.Fatalf("UpdateActionConfig: %v", err)
	}
	if ac.Name != newName {
		t.Errorf("expected name %q, got %q", newName, ac.Name)
	}
	// Other fields should be unchanged
	if ac.Status != "active" {
		t.Errorf("expected status to remain 'active', got %q", ac.Status)
	}
	if ac.Description == nil || *ac.Description != desc {
		t.Errorf("expected description to remain %q, got %v", desc, ac.Description)
	}
}

func TestUpdateActionConfig_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	newName := "Nope"
	_, err := db.UpdateActionConfig(t.Context(), tx, db.UpdateActionConfigParams{
		ID:     "nonexistent",
		UserID: uid,
		Name:   &newName,
	})
	if err == nil {
		t.Fatal("expected error for non-existent config")
	}
	acErr, ok := err.(*db.ActionConfigError)
	if !ok {
		t.Fatalf("expected *ActionConfigError, got %T: %v", err, err)
	}
	if acErr.Code != db.ActionConfigErrNotFound {
		t.Errorf("expected ActionConfigErrNotFound, got %v", acErr.Code)
	}
}

func TestUpdateActionConfig_WrongUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid1, connID, "test.action")

	// User 2 tries to update User 1's config
	newName := "Hacked"
	_, err := db.UpdateActionConfig(t.Context(), tx, db.UpdateActionConfigParams{
		ID:     configID,
		UserID: uid2,
		Name:   &newName,
	})
	if err == nil {
		t.Fatal("expected error for wrong user")
	}
	acErr, ok := err.(*db.ActionConfigError)
	if !ok {
		t.Fatalf("expected *ActionConfigError, got %T: %v", err, err)
	}
	if acErr.Code != db.ActionConfigErrNotFound {
		t.Errorf("expected ActionConfigErrNotFound, got %v", acErr.Code)
	}
}

// ── GetActiveActionConfigForAgent ─────────────────────────────────────────────

func TestGetActiveActionConfigForAgent_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, "test.action")

	ac, err := db.GetActiveActionConfigForAgent(t.Context(), tx, configID, agentID)
	if err != nil {
		t.Fatalf("GetActiveActionConfigForAgent: %v", err)
	}
	if ac == nil {
		t.Fatal("expected non-nil result")
	}
	if ac.ID != configID {
		t.Errorf("expected id %q, got %q", configID, ac.ID)
	}
}

func TestGetActiveActionConfigForAgent_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	ac, err := db.GetActiveActionConfigForAgent(t.Context(), tx, "nonexistent", agentID)
	if err != nil {
		t.Fatalf("GetActiveActionConfigForAgent: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for non-existent config")
	}
}

func TestGetActiveActionConfigForAgent_DisabledConfig(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, "test.action", testhelper.ActionConfigOpts{
		Status: "disabled",
		Name:   "Disabled",
	})

	ac, err := db.GetActiveActionConfigForAgent(t.Context(), tx, configID, agentID)
	if err != nil {
		t.Fatalf("GetActiveActionConfigForAgent: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for disabled config")
	}
}

func TestGetActiveActionConfigForAgent_DeactivatedAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	// Agent exists but is not registered (pending status).
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, "test.action")

	ac, err := db.GetActiveActionConfigForAgent(t.Context(), tx, configID, agentID)
	if err != nil {
		t.Fatalf("GetActiveActionConfigForAgent: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for non-registered agent")
	}
}

func TestGetActiveActionConfigForAgent_WrongAgent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agentID1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	agentID2 := testhelper.InsertUserWithAgent(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	// Config belongs to agent 1
	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID1, uid1, connID, "test.action")

	// Agent 2 should NOT see agent 1's config
	ac, err := db.GetActiveActionConfigForAgent(t.Context(), tx, configID, agentID2)
	if err != nil {
		t.Fatalf("GetActiveActionConfigForAgent: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for config belonging to different agent")
	}
}

// ── DeleteActionConfig ───────────────────────────────────────────────────────

func TestDeleteActionConfig_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid, connID, "test.action")

	ac, err := db.DeleteActionConfig(t.Context(), tx, configID, uid)
	if err != nil {
		t.Fatalf("DeleteActionConfig: %v", err)
	}
	if ac == nil {
		t.Fatal("expected non-nil result")
	}
	if ac.ID != configID {
		t.Errorf("expected id %q, got %q", configID, ac.ID)
	}

	// Verify it was actually deleted
	got, err := db.GetActionConfigByID(t.Context(), tx, configID, uid)
	if err != nil {
		t.Fatalf("GetActionConfigByID after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestDeleteActionConfig_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	ac, err := db.DeleteActionConfig(t.Context(), tx, "nonexistent", uid)
	if err != nil {
		t.Fatalf("DeleteActionConfig: %v", err)
	}
	if ac != nil {
		t.Error("expected nil for non-existent config")
	}
}

func TestDeleteActionConfig_WrongUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid1 := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid1, "u1_"+uid1[:6])

	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:6])

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfig(t, tx, configID, agentID, uid1, connID, "test.action")

	// User 2 tries to delete User 1's config
	ac, err := db.DeleteActionConfig(t.Context(), tx, configID, uid2)
	if err != nil {
		t.Fatalf("DeleteActionConfig: %v", err)
	}
	if ac != nil {
		t.Error("expected nil when deleting as wrong user")
	}

	// Verify it still exists for User 1
	got, err := db.GetActionConfigByID(t.Context(), tx, configID, uid1)
	if err != nil {
		t.Fatalf("GetActionConfigByID: %v", err)
	}
	if got == nil {
		t.Error("config should still exist after failed delete by wrong user")
	}
}

// ── Wildcard Action Type ────────────────────────────────────────────────────

func TestWildcardActionType_InsertSuccess(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	// Wildcard config — no matching connector_action row needed
	configID := testhelper.GenerateID(t, "ac_")
	testhelper.InsertActionConfigFull(t, tx, configID, agentID, uid, connID, "*", testhelper.ActionConfigOpts{
		Name: "All Actions",
	})

	ac, err := db.GetActionConfigByID(t.Context(), tx, configID, uid)
	if err != nil {
		t.Fatalf("GetActionConfigByID: %v", err)
	}
	if ac == nil {
		t.Fatal("expected non-nil result for wildcard config")
	}
	if ac.ActionType != "*" {
		t.Errorf("expected action_type '*', got %q", ac.ActionType)
	}
}

func TestWildcardActionType_UniquenessConstraint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	// First wildcard config succeeds
	testhelper.InsertActionConfigFull(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, "*", testhelper.ActionConfigOpts{
		Name: "All Actions 1",
	})

	// Second wildcard config for same agent+connector should fail
	err := testhelper.WithSavepoint(t, tx, func() error {
		_, err := tx.Exec(context.Background(),
			`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, name)
			 VALUES ($1, $2, $3, $4, '*', '{}', 'All Actions 2')`,
			testhelper.GenerateID(t, "ac_"), agentID, uid, connID)
		return err
	})
	if err == nil {
		t.Error("expected unique constraint violation for duplicate wildcard config, but insert succeeded")
	}
}

func TestWildcardActionType_AllowedAlongsideSpecific(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	// Wildcard config
	testhelper.InsertActionConfigFull(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, "*", testhelper.ActionConfigOpts{
		Name: "All Actions",
	})

	// Specific config for the same agent+connector should also work
	testhelper.InsertActionConfig(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, connID, "test.action")

	configs, err := db.ListActionConfigsByAgent(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("ListActionConfigsByAgent: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 configs (wildcard + specific), got %d", len(configs))
	}
}

func TestWildcardActionType_UniquePerConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)

	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	conn1 := testhelper.GenerateID(t, "conn_")
	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn1)
	testhelper.InsertConnector(t, tx, conn2)

	// Wildcard config for connector 1
	testhelper.InsertActionConfigFull(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, conn1, "*", testhelper.ActionConfigOpts{
		Name: "All Conn1 Actions",
	})

	// Wildcard config for connector 2 — should succeed (different connector)
	testhelper.InsertActionConfigFull(t, tx, testhelper.GenerateID(t, "ac_"), agentID, uid, conn2, "*", testhelper.ActionConfigOpts{
		Name: "All Conn2 Actions",
	})

	configs, err := db.ListActionConfigsByAgent(t.Context(), tx, agentID, uid)
	if err != nil {
		t.Fatalf("ListActionConfigsByAgent: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 wildcard configs (one per connector), got %d", len(configs))
	}
}

// ── ConnectorActionExists ────────────────────────────────────────────────────

func TestConnectorActionExists_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	exists, err := db.ConnectorActionExists(t.Context(), tx, connID, "test.action")
	if err != nil {
		t.Fatalf("ConnectorActionExists: %v", err)
	}
	if !exists {
		t.Error("expected connector action to exist")
	}
}

func TestConnectorActionExists_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	exists, err := db.ConnectorActionExists(t.Context(), tx, connID, "nonexistent")
	if err != nil {
		t.Fatalf("ConnectorActionExists: %v", err)
	}
	if exists {
		t.Error("expected connector action to not exist")
	}
}

func TestWildcardActionType_IndexExists(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "action_configurations", "idx_action_config_wildcard_unique")
}
