package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestConnectorsSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	t.Run("connectors", func(t *testing.T) {
		testhelper.RequireColumns(t, tx, "connectors", []string{"id", "name", "description", "created_at"})
	})
	t.Run("connector_actions", func(t *testing.T) {
		testhelper.RequireColumns(t, tx, "connector_actions", []string{
			"connector_id", "action_type", "name", "description",
			"risk_level", "parameters_schema",
		})
	})
	t.Run("connector_required_credentials", func(t *testing.T) {
		testhelper.RequireColumns(t, tx, "connector_required_credentials", []string{
			"connector_id", "service", "auth_type", "instructions_url",
			"oauth_provider", "oauth_scopes",
		})
	})
	t.Run("oauth_connections", func(t *testing.T) {
		testhelper.RequireColumns(t, tx, "oauth_connections", []string{
			"id", "user_id", "provider", "access_token_vault_id",
			"refresh_token_vault_id", "scopes", "token_expiry",
			"status", "created_at", "updated_at",
		})
	})
	t.Run("oauth_provider_configs", func(t *testing.T) {
		testhelper.RequireColumns(t, tx, "oauth_provider_configs", []string{
			"id", "user_id", "provider", "client_id_vault_id",
			"client_secret_vault_id", "created_at", "updated_at",
		})
	})
}

func TestConnectorCascadeDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	connID := testhelper.GenerateID(t, "conn_")
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "send", "Send")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "api", "api_key")

	testhelper.RequireCascadeDeletes(t, tx,
		"DELETE FROM connectors WHERE id = '"+connID+"'",
		[]string{"connector_actions", "connector_required_credentials"},
		"connector_id = '"+connID+"'",
	)
}

func TestGetRequiredServicesByActionType_Found(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.InsertConnector(t, tx, "github")
	testhelper.InsertConnectorAction(t, tx, "github", "github.create_issue", "Create Issue")
	testhelper.InsertConnectorRequiredCredential(t, tx, "github", "github", "api_key")

	services, err := db.GetRequiredServicesByActionType(context.Background(), tx, "github.create_issue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0] != "github" {
		t.Errorf("expected service 'github', got %q", services[0])
	}
}

func TestGetRequiredServicesByActionType_NoCredentials(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.InsertConnector(t, tx, "nocred")
	testhelper.InsertConnectorAction(t, tx, "nocred", "nocred.ping", "Ping")

	services, err := db.GetRequiredServicesByActionType(context.Background(), tx, "nocred.ping")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 0 {
		t.Fatalf("expected 0 services, got %d: %v", len(services), services)
	}
}

func TestGetRequiredServicesByActionType_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	services, err := db.GetRequiredServicesByActionType(context.Background(), tx, "nonexistent.action")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if services != nil {
		t.Fatalf("expected nil for not-found action type, got %v", services)
	}
}

func TestGetRequiredServicesByActionType_MultipleServices(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.InsertConnector(t, tx, "multi")
	testhelper.InsertConnectorAction(t, tx, "multi", "multi.sync", "Sync")
	testhelper.InsertConnectorRequiredCredential(t, tx, "multi", "github", "api_key")
	testhelper.InsertConnectorRequiredCredential(t, tx, "multi", "slack", "api_key")

	services, err := db.GetRequiredServicesByActionType(context.Background(), tx, "multi.sync")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d: %v", len(services), services)
	}
	// Should be ordered by service.
	if services[0] != "github" || services[1] != "slack" {
		t.Errorf("expected [github, slack], got %v", services)
	}
}

func TestListConnectorIDs_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	ids, err := db.ListConnectorIDs(context.Background(), tx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected 0 IDs, got %d: %v", len(ids), ids)
	}
}

func TestDeleteConnectorByID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.InsertConnector(t, tx, "gone")
	if err := db.DeleteConnectorByID(context.Background(), tx, "gone"); err != nil {
		t.Fatalf("DeleteConnectorByID: %v", err)
	}
	ids, err := db.ListConnectorIDs(context.Background(), tx)
	if err != nil {
		t.Fatalf("ListConnectorIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected connector deleted, still have %v", ids)
	}
}

func TestListConnectorIDs_ReturnsAll(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.InsertConnector(t, tx, "github")
	testhelper.InsertConnector(t, tx, "slack")

	ids, err := db.ListConnectorIDs(context.Background(), tx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d: %v", len(ids), ids)
	}
	// Ordered by id.
	if ids[0] != "github" || ids[1] != "slack" {
		t.Errorf("expected [github, slack], got %v", ids)
	}
}

func TestUpsertConnectorFromManifest_Insert(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	m := db.ExternalConnectorManifest{
		ID:          "ext-test",
		Name:        "External Test",
		Description: "A test connector",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-test.create", Name: "Create", Description: "Create thing", RiskLevel: "low"},
			{ActionType: "ext-test.delete", Name: "Delete", RiskLevel: "high"},
		},
		Credentials: []db.ExternalConnectorCredential{
			{Service: "ext-test", AuthType: "api_key"},
		},
	}

	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify connector row.
	var name, desc string
	err := tx.QueryRow(ctx, "SELECT name, COALESCE(description, '') FROM connectors WHERE id = $1", "ext-test").Scan(&name, &desc)
	if err != nil {
		t.Fatalf("querying connector: %v", err)
	}
	if name != "External Test" {
		t.Errorf("name = %q, want %q", name, "External Test")
	}
	if desc != "A test connector" {
		t.Errorf("description = %q, want %q", desc, "A test connector")
	}

	// Verify actions.
	var actionCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM connector_actions WHERE connector_id = $1", "ext-test").Scan(&actionCount)
	if err != nil {
		t.Fatalf("counting actions: %v", err)
	}
	if actionCount != 2 {
		t.Errorf("action count = %d, want 2", actionCount)
	}

	// Verify credentials.
	var credCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM connector_required_credentials WHERE connector_id = $1", "ext-test").Scan(&credCount)
	if err != nil {
		t.Fatalf("counting credentials: %v", err)
	}
	if credCount != 1 {
		t.Errorf("credential count = %d, want 1", credCount)
	}
}

func TestUpsertConnectorFromManifest_Update(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	// Insert initial version.
	m := db.ExternalConnectorManifest{
		ID:   "ext-upd",
		Name: "Original Name",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-upd.action", Name: "Original Action", RiskLevel: "low"},
		},
		Credentials: []db.ExternalConnectorCredential{
			{Service: "ext-upd", AuthType: "api_key"},
		},
	}
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Update with new name and action name.
	m.Name = "Updated Name"
	m.Actions[0].Name = "Updated Action"
	m.Actions[0].RiskLevel = "high"
	m.Credentials[0].AuthType = "basic"

	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("update upsert: %v", err)
	}

	// Verify connector name was updated.
	var name string
	err := tx.QueryRow(ctx, "SELECT name FROM connectors WHERE id = $1", "ext-upd").Scan(&name)
	if err != nil {
		t.Fatalf("querying connector: %v", err)
	}
	if name != "Updated Name" {
		t.Errorf("name = %q, want %q", name, "Updated Name")
	}

	// Verify action was updated (not duplicated).
	var actionName, riskLevel string
	err = tx.QueryRow(ctx,
		"SELECT name, COALESCE(risk_level, '') FROM connector_actions WHERE connector_id = $1 AND action_type = $2",
		"ext-upd", "ext-upd.action").Scan(&actionName, &riskLevel)
	if err != nil {
		t.Fatalf("querying action: %v", err)
	}
	if actionName != "Updated Action" {
		t.Errorf("action name = %q, want %q", actionName, "Updated Action")
	}
	if riskLevel != "high" {
		t.Errorf("risk_level = %q, want %q", riskLevel, "high")
	}

	// Verify credential auth_type was updated.
	var authType string
	err = tx.QueryRow(ctx,
		"SELECT auth_type FROM connector_required_credentials WHERE connector_id = $1 AND service = $2",
		"ext-upd", "ext-upd").Scan(&authType)
	if err != nil {
		t.Fatalf("querying credential: %v", err)
	}
	if authType != "basic" {
		t.Errorf("auth_type = %q, want %q", authType, "basic")
	}
}

func TestUpsertConnectorFromManifest_RemovesStaleRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	// Insert with two actions and two credentials.
	m := db.ExternalConnectorManifest{
		ID:   "ext-stale",
		Name: "Stale Test",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-stale.a", Name: "Action A"},
			{ActionType: "ext-stale.b", Name: "Action B"},
		},
		Credentials: []db.ExternalConnectorCredential{
			{Service: "svc-one", AuthType: "api_key"},
			{Service: "svc-two", AuthType: "basic"},
		},
	}
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Re-upsert with only one action and one credential — the others should be deleted.
	m.Actions = []db.ExternalConnectorAction{
		{ActionType: "ext-stale.a", Name: "Action A Updated"},
	}
	m.Credentials = []db.ExternalConnectorCredential{
		{Service: "svc-one", AuthType: "api_key"},
	}
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	// Verify stale action was removed.
	var actionCount int
	err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connector_actions WHERE connector_id = $1", "ext-stale").Scan(&actionCount)
	if err != nil {
		t.Fatalf("counting actions: %v", err)
	}
	if actionCount != 1 {
		t.Errorf("action count = %d, want 1 (stale action should be removed)", actionCount)
	}

	// Verify stale credential was removed.
	var credCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM connector_required_credentials WHERE connector_id = $1", "ext-stale").Scan(&credCount)
	if err != nil {
		t.Fatalf("counting credentials: %v", err)
	}
	if credCount != 1 {
		t.Errorf("credential count = %d, want 1 (stale credential should be removed)", credCount)
	}
}

func TestUpsertConnectorFromManifest_NoCredentials(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	m := db.ExternalConnectorManifest{
		ID:   "ext-nocred",
		Name: "No Creds",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-nocred.ping", Name: "Ping"},
		},
	}

	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var credCount int
	err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM connector_required_credentials WHERE connector_id = $1", "ext-nocred").Scan(&credCount)
	if err != nil {
		t.Fatalf("counting credentials: %v", err)
	}
	if credCount != 0 {
		t.Errorf("credential count = %d, want 0", credCount)
	}
}

func TestUpsertConnectorFromManifest_InstructionsURL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	wantURL := "https://docs.example.com/setup"
	m := db.ExternalConnectorManifest{
		ID:   "ext-url",
		Name: "URL Test",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-url.ping", Name: "Ping"},
		},
		Credentials: []db.ExternalConnectorCredential{
			{Service: "ext-url", AuthType: "api_key", InstructionsURL: wantURL},
		},
	}

	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Verify via GetConnectorByID.
	detail, err := db.GetConnectorByID(ctx, tx, "ext-url")
	if err != nil {
		t.Fatalf("GetConnectorByID: %v", err)
	}
	if detail == nil {
		t.Fatal("GetConnectorByID returned nil")
	}
	if len(detail.RequiredCredentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(detail.RequiredCredentials))
	}
	cred := detail.RequiredCredentials[0]
	if cred.InstructionsURL == nil || *cred.InstructionsURL != wantURL {
		t.Errorf("instructions_url = %v, want %q", cred.InstructionsURL, wantURL)
	}
}

func TestUpsertConnectorFromManifest_InstructionsURLNull(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	m := db.ExternalConnectorManifest{
		ID:   "ext-nourl",
		Name: "No URL Test",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-nourl.ping", Name: "Ping"},
		},
		Credentials: []db.ExternalConnectorCredential{
			{Service: "ext-nourl", AuthType: "api_key"},
		},
	}

	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	detail, err := db.GetConnectorByID(ctx, tx, "ext-nourl")
	if err != nil {
		t.Fatalf("GetConnectorByID: %v", err)
	}
	if detail == nil {
		t.Fatal("GetConnectorByID returned nil")
	}
	if len(detail.RequiredCredentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(detail.RequiredCredentials))
	}
	if detail.RequiredCredentials[0].InstructionsURL != nil {
		t.Errorf("instructions_url = %v, want nil", detail.RequiredCredentials[0].InstructionsURL)
	}
}

// ── Template Upsert Tests ────────────────────────────────────────────────────

func TestUpsertConnectorFromManifest_WithTemplates(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	m := db.ExternalConnectorManifest{
		ID:   "ext-tpl",
		Name: "Template Test",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-tpl.create", Name: "Create"},
		},
		Templates: []db.ExternalConnectorTemplate{
			{
				ID:          "tpl_ext_create",
				ActionType:  "ext-tpl.create",
				Name:        "Default create",
				Description: "Pre-filled template",
				Parameters:  []byte(`{"key":"*"}`),
			},
		},
	}

	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Verify template was created.
	templates, err := db.ListTemplatesByConnector(ctx, tx, "ext-tpl")
	if err != nil {
		t.Fatalf("ListTemplatesByConnector: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
	if templates[0].ID != "tpl_ext_create" {
		t.Errorf("template ID = %q, want %q", templates[0].ID, "tpl_ext_create")
	}
	if templates[0].Name != "Default create" {
		t.Errorf("template name = %q, want %q", templates[0].Name, "Default create")
	}
}

func TestUpsertConnectorFromManifest_TemplateUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	m := db.ExternalConnectorManifest{
		ID:   "ext-tplupd",
		Name: "Template Update",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-tplupd.do", Name: "Do"},
		},
		Templates: []db.ExternalConnectorTemplate{
			{ID: "tpl_upd", ActionType: "ext-tplupd.do", Name: "Original", Parameters: []byte(`{"a":"*"}`)},
		},
	}
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Update template name and parameters.
	m.Templates[0].Name = "Updated"
	m.Templates[0].Parameters = []byte(`{"a":"fixed"}`)
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("update upsert: %v", err)
	}

	templates, err := db.ListTemplatesByConnector(ctx, tx, "ext-tplupd")
	if err != nil {
		t.Fatalf("ListTemplatesByConnector: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template, got %d", len(templates))
	}
	if templates[0].Name != "Updated" {
		t.Errorf("template name = %q, want %q", templates[0].Name, "Updated")
	}
}

func TestUpsertConnectorFromManifest_RemovesStaleTemplates(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	m := db.ExternalConnectorManifest{
		ID:   "ext-tplstale",
		Name: "Stale Templates",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-tplstale.do", Name: "Do"},
		},
		Templates: []db.ExternalConnectorTemplate{
			{ID: "tpl_keep", ActionType: "ext-tplstale.do", Name: "Keep", Parameters: []byte(`{"a":"*"}`)},
			{ID: "tpl_remove", ActionType: "ext-tplstale.do", Name: "Remove", Parameters: []byte(`{"b":"*"}`)},
		},
	}
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Re-upsert with only one template — the other should be removed.
	m.Templates = []db.ExternalConnectorTemplate{
		{ID: "tpl_keep", ActionType: "ext-tplstale.do", Name: "Keep", Parameters: []byte(`{"a":"*"}`)},
	}
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	templates, err := db.ListTemplatesByConnector(ctx, tx, "ext-tplstale")
	if err != nil {
		t.Fatalf("ListTemplatesByConnector: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 template (stale removed), got %d", len(templates))
	}
	if templates[0].ID != "tpl_keep" {
		t.Errorf("template ID = %q, want %q", templates[0].ID, "tpl_keep")
	}
}

func TestUpsertConnectorFromManifest_NoTemplates(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	// Insert with templates.
	m := db.ExternalConnectorManifest{
		ID:   "ext-tplnone",
		Name: "No Templates",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-tplnone.do", Name: "Do"},
		},
		Templates: []db.ExternalConnectorTemplate{
			{ID: "tpl_will_remove", ActionType: "ext-tplnone.do", Name: "T", Parameters: []byte(`{"a":"*"}`)},
		},
	}
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Re-upsert with no templates — existing ones should be removed.
	m.Templates = nil
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	templates, err := db.ListTemplatesByConnector(ctx, tx, "ext-tplnone")
	if err != nil {
		t.Fatalf("ListTemplatesByConnector: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("expected 0 templates after removal, got %d", len(templates))
	}
}

func TestUpsertConnectorFromManifest_InstructionsURLUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tx := testhelper.SetupTestDB(t)

	// Insert with no URL.
	m := db.ExternalConnectorManifest{
		ID:   "ext-urlupd",
		Name: "URL Update",
		Actions: []db.ExternalConnectorAction{
			{ActionType: "ext-urlupd.ping", Name: "Ping"},
		},
		Credentials: []db.ExternalConnectorCredential{
			{Service: "ext-urlupd", AuthType: "api_key"},
		},
	}
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Update with a URL.
	wantURL := "https://docs.example.com/new-setup"
	m.Credentials[0].InstructionsURL = wantURL
	if err := db.UpsertConnectorFromManifest(ctx, tx, m); err != nil {
		t.Fatalf("update upsert: %v", err)
	}

	detail, err := db.GetConnectorByID(ctx, tx, "ext-urlupd")
	if err != nil {
		t.Fatalf("GetConnectorByID: %v", err)
	}
	if detail == nil {
		t.Fatal("GetConnectorByID returned nil")
	}
	cred := detail.RequiredCredentials[0]
	if cred.InstructionsURL == nil || *cred.InstructionsURL != wantURL {
		t.Errorf("instructions_url = %v, want %q", cred.InstructionsURL, wantURL)
	}
}
