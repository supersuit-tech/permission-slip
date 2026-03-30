package db_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── ListConnectors ──────────────────────────────────────────────────────────

func TestListConnectors_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connectors, err := db.ListConnectors(t.Context(), tx)
	if err != nil {
		t.Fatalf("ListConnectors: %v", err)
	}
	if len(connectors) != 0 {
		t.Errorf("expected 0 connectors, got %d", len(connectors))
	}
}

func TestListConnectors_WithData(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action1", "Action 1")
	testhelper.InsertConnectorAction(t, tx, connID, "test.action2", "Action 2")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "test_svc", "api_key")

	connectors, err := db.ListConnectors(t.Context(), tx)
	if err != nil {
		t.Fatalf("ListConnectors: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}

	c := connectors[0]
	if c.ID != connID {
		t.Errorf("expected id %q, got %q", connID, c.ID)
	}
	if c.Name != connID {
		t.Errorf("expected name %q, got %q", connID, c.Name)
	}
	if len(c.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(c.Actions))
	}
	if len(c.RequiredCredentials) != 1 {
		t.Errorf("expected 1 required credential, got %d", len(c.RequiredCredentials))
	}
	if c.RequiredCredentials[0] != "test_svc" {
		t.Errorf("expected required credential 'test_svc', got %q", c.RequiredCredentials[0])
	}
}

func TestListConnectors_MultipleConnectors(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	conn1 := testhelper.GenerateID(t, "conn_")
	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn1)
	testhelper.InsertConnector(t, tx, conn2)

	connectors, err := db.ListConnectors(t.Context(), tx)
	if err != nil {
		t.Fatalf("ListConnectors: %v", err)
	}
	if len(connectors) != 2 {
		t.Fatalf("expected 2 connectors, got %d", len(connectors))
	}
}

func TestListConnectors_NoActionsOrCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)

	connectors, err := db.ListConnectors(t.Context(), tx)
	if err != nil {
		t.Fatalf("ListConnectors: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 connector, got %d", len(connectors))
	}
	if len(connectors[0].Actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(connectors[0].Actions))
	}
	if len(connectors[0].RequiredCredentials) != 0 {
		t.Errorf("expected 0 required credentials, got %d", len(connectors[0].RequiredCredentials))
	}
}

// ── GetConnectorByID ────────────────────────────────────────────────────────

func TestGetConnectorByID_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.act", "Test Act")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "svc", "api_key")

	connector, err := db.GetConnectorByID(t.Context(), tx, connID)
	if err != nil {
		t.Fatalf("GetConnectorByID: %v", err)
	}
	if connector == nil {
		t.Fatal("expected connector, got nil")
	}
	if connector.ID != connID {
		t.Errorf("expected id %q, got %q", connID, connector.ID)
	}
	if len(connector.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(connector.Actions))
	}
	if connector.Actions[0].ActionType != "test.act" {
		t.Errorf("expected action_type 'test.act', got %q", connector.Actions[0].ActionType)
	}
	if connector.Actions[0].Name != "Test Act" {
		t.Errorf("expected action name 'Test Act', got %q", connector.Actions[0].Name)
	}
	if len(connector.RequiredCredentials) != 1 {
		t.Fatalf("expected 1 required credential, got %d", len(connector.RequiredCredentials))
	}
	if connector.RequiredCredentials[0].Service != "svc" {
		t.Errorf("expected service 'svc', got %q", connector.RequiredCredentials[0].Service)
	}
	if connector.RequiredCredentials[0].AuthType != "api_key" {
		t.Errorf("expected auth_type 'api_key', got %q", connector.RequiredCredentials[0].AuthType)
	}
}

func TestGetConnectorByID_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connector, err := db.GetConnectorByID(t.Context(), tx, "nonexistent")
	if err != nil {
		t.Fatalf("GetConnectorByID: %v", err)
	}
	if connector != nil {
		t.Error("expected nil for nonexistent connector")
	}
}
