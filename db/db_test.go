package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestDatabaseConnectivity(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	var result int
	err := tx.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}
}

func TestMigrationsApplied(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	var uuidExists bool
	err := tx.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp')").Scan(&uuidExists)
	if err != nil {
		t.Fatalf("failed to check uuid-ossp extension: %v", err)
	}
	if !uuidExists {
		t.Fatal("uuid-ossp extension should be installed after migrations")
	}

	var pgcryptoExists bool
	err = tx.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto')").Scan(&pgcryptoExists)
	if err != nil {
		t.Fatalf("failed to check pgcrypto extension: %v", err)
	}
	if !pgcryptoExists {
		t.Fatal("pgcrypto extension should be installed after migrations")
	}
}
