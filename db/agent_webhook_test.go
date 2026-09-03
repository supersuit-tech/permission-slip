package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestAgentWebhook_ProviderRoundTrip(t *testing.T) {
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	if err := db.SetAgentWebhook(ctx, tx, agentID, "https://api2.cursor.sh/automations/webhook/wh_x", "vault-1", "grokbot"); err != nil {
		t.Fatal(err)
	}
	cfg, err := db.GetAgentWebhookConfig(ctx, tx, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || cfg.WebhookURL == nil || *cfg.WebhookURL == "" {
		t.Fatal("expected webhook url")
	}
	if cfg.WebhookProvider != "grokbot" {
		t.Fatalf("provider = %q", cfg.WebhookProvider)
	}

	prev, err := db.ClearAgentWebhook(ctx, tx, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if prev == nil || *prev != "vault-1" {
		t.Fatalf("prev vault = %v", prev)
	}
	cfg, err = db.GetAgentWebhookConfig(ctx, tx, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WebhookURL != nil {
		t.Fatal("expected url cleared")
	}
	if cfg.WebhookProvider != "openclaw" {
		t.Fatalf("cleared provider = %q, want openclaw default", cfg.WebhookProvider)
	}
}
