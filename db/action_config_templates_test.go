package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

// ── Schema Tests ─────────────────────────────────────────────────────────────

func TestActionConfigTemplatesSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "action_config_templates", []string{
		"id", "connector_id", "action_type", "name", "description",
		"parameters", "created_at",
	})
}

func TestActionConfigTemplatesIndexConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireIndex(t, tx, "action_config_templates", "idx_action_config_templates_connector")
}

// ── FK Cascade Tests ─────────────────────────────────────────────────────────

func TestActionConfigTemplatesCascadeOnConnectorDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	testhelper.InsertActionConfigTemplate(t, tx, "tpl_cascade_test", connID, "test.action", "Test Template")

	// Delete the connector — template should cascade.
	_, err := tx.Exec(context.Background(), `DELETE FROM connectors WHERE id = $1`, connID)
	if err != nil {
		t.Fatalf("delete connector: %v", err)
	}

	var count int
	if err := tx.QueryRow(context.Background(),
		`SELECT count(*) FROM action_config_templates WHERE id = $1`,
		"tpl_cascade_test").Scan(&count); err != nil {
		t.Fatalf("count check: %v", err)
	}
	if count != 0 {
		t.Errorf("expected template to be cascade-deleted, got count=%d", count)
	}
}

func TestActionConfigTemplatesCascadeOnActionDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "test.action", "Test Action")

	testhelper.InsertActionConfigTemplate(t, tx, "tpl_action_cascade", connID, "test.action", "Template")

	// Delete the action — template should cascade via FK on (connector_id, action_type).
	_, err := tx.Exec(context.Background(),
		`DELETE FROM connector_actions WHERE connector_id = $1 AND action_type = $2`,
		connID, "test.action")
	if err != nil {
		t.Fatalf("delete connector action: %v", err)
	}

	var count int
	if err := tx.QueryRow(context.Background(),
		`SELECT count(*) FROM action_config_templates WHERE id = $1`,
		"tpl_action_cascade").Scan(&count); err != nil {
		t.Fatalf("count check: %v", err)
	}
	if count != 0 {
		t.Errorf("expected template to be cascade-deleted, got count=%d", count)
	}
}

// ── ListTemplatesByConnector Tests ───────────────────────────────────────────

func TestListTemplatesByConnector_ReturnsTemplatesForConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	conn1 := testhelper.GenerateID(t, "conn_")
	conn2 := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, conn1)
	testhelper.InsertConnector(t, tx, conn2)
	testhelper.InsertConnectorAction(t, tx, conn1, "c1.action_a", "Action A")
	testhelper.InsertConnectorAction(t, tx, conn1, "c1.action_b", "Action B")
	testhelper.InsertConnectorAction(t, tx, conn2, "c2.action_a", "Other Action")

	desc := "A test description"
	testhelper.InsertActionConfigTemplateFull(t, tx, "tpl_1", conn1, "c1.action_a", "Template 1", testhelper.ActionConfigTemplateOpts{
		Description: &desc,
		Parameters:  []byte(`{"repo": "*", "title": "*"}`),
	})
	testhelper.InsertActionConfigTemplate(t, tx, "tpl_2", conn1, "c1.action_b", "Template 2")
	testhelper.InsertActionConfigTemplate(t, tx, "tpl_other", conn2, "c2.action_a", "Other Template")

	// List for conn1 — should return 2 templates, ordered by action_type.
	templates, err := db.ListTemplatesByConnector(context.Background(), tx, conn1)
	if err != nil {
		t.Fatalf("ListTemplatesByConnector: %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(templates))
	}
	if templates[0].ID != "tpl_1" {
		t.Errorf("expected first template ID 'tpl_1', got %q", templates[0].ID)
	}
	if templates[0].Description == nil || *templates[0].Description != desc {
		t.Errorf("expected description %q, got %v", desc, templates[0].Description)
	}
	if templates[1].ID != "tpl_2" {
		t.Errorf("expected second template ID 'tpl_2', got %q", templates[1].ID)
	}

	// List for conn2 — should return 1 template.
	templates2, err := db.ListTemplatesByConnector(context.Background(), tx, conn2)
	if err != nil {
		t.Fatalf("ListTemplatesByConnector for conn2: %v", err)
	}
	if len(templates2) != 1 {
		t.Fatalf("expected 1 template for conn2, got %d", len(templates2))
	}
}

func TestListTemplatesByConnector_EmptyForUnknownConnector(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	templates, err := db.ListTemplatesByConnector(context.Background(), tx, "nonexistent")
	if err != nil {
		t.Fatalf("ListTemplatesByConnector: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected 0 templates, got %d", len(templates))
	}
}
