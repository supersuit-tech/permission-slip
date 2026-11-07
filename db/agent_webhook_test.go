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

	if err := db.SetAgentWakeCredentials(ctx, tx, agentID, "vault-1", "grokbot"); err != nil {
		t.Fatal(err)
	}
	cfg, err := db.GetAgentWebhookConfig(ctx, tx, agentID)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || cfg.WebhookTokenVaultID == nil || *cfg.WebhookTokenVaultID != "vault-1" {
		t.Fatal("expected wake token vault id")
	}
	if cfg.WebhookURL != nil {
		t.Fatal("wake registration must not store a destination URL")
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
