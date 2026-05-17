package testhelper

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// TestDatabasePath returns the SQLite path used for tests. Defaults to a
// shared in-memory database so tests run fast and need no filesystem setup.
// Override with DATABASE_PATH_TEST if you want a real file (useful for
// debugging a failing test's state with the sqlite3 CLI).
func TestDatabasePath() string {
	if p := os.Getenv("DATABASE_PATH_TEST"); p != "" {
		return p
	}
	return ":memory:"
}

var (
	sharedPool     *db.Pool
	sharedPoolOnce sync.Once
	sharedPoolErr  error
)

// getSharedPool returns a connection pool shared across all tests in a binary.
// Migrations run exactly once on first access.
func getSharedPool(t *testing.T) *db.Pool {
	t.Helper()

	sharedPoolOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		path := TestDatabasePath()

		var err error
		sharedPool, err = db.Connect(ctx, path)
		if err != nil {
			sharedPoolErr = fmt.Errorf("connect test database: %w", err)
			return
		}
		if err := db.MigratePool(ctx, sharedPool); err != nil {
			sharedPoolErr = fmt.Errorf("migrate test database: %w", err)
			_ = sharedPool.Close()
			sharedPool = nil
			return
		}
	})
	if sharedPoolErr != nil {
		t.Fatalf("failed to initialize shared test pool: %v", sharedPoolErr)
	}
	return sharedPool
}

// SetupTestDB returns a db.DBTX backed by a transaction that is automatically
// rolled back when the test completes. This provides complete data isolation:
// each test's inserts/updates/deletes are invisible to other tests and leave
// no residue in the database.
//
// Note: with SQLite's single-writer model, tests using SetupTestDB should NOT
// call t.Parallel() — concurrent write transactions on the shared in-memory
// DB will serialize and the tests may flake on timing.
func SetupTestDB(t *testing.T) db.DBTX {
	t.Helper()

	pool := getSharedPool(t)
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin test transaction: %v", err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback()
	})
	return tx
}

// SetupPool returns the shared *db.Pool for tests that need raw pool access
// (e.g., tests that manage their own transactions or test pool-level behavior).
func SetupPool(t *testing.T) *db.Pool {
	t.Helper()
	return getSharedPool(t)
}
