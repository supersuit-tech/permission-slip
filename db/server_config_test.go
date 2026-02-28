package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestServerConfigSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.RequireColumns(t, tx, "server_config", []string{"key", "value", "created_at", "updated_at"})
}

func TestGetServerConfig_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	val, err := db.GetServerConfig(context.Background(), tx, "nonexistent_key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string for missing key, got %q", val)
	}
}

func TestSetAndGetServerConfig(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	err := db.SetServerConfig(ctx, tx, "test_key", "test_value")
	if err != nil {
		t.Fatalf("set: %v", err)
	}

	val, err := db.GetServerConfig(ctx, tx, "test_key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "test_value" {
		t.Errorf("expected 'test_value', got %q", val)
	}
}

func TestSetServerConfig_Upsert(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	if err := db.SetServerConfig(ctx, tx, "test_key", "original"); err != nil {
		t.Fatalf("first set: %v", err)
	}

	if err := db.SetServerConfig(ctx, tx, "test_key", "updated"); err != nil {
		t.Fatalf("second set: %v", err)
	}

	val, err := db.GetServerConfig(ctx, tx, "test_key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "updated" {
		t.Errorf("expected 'updated', got %q", val)
	}
}
