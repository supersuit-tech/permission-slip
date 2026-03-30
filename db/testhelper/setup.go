package testhelper

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/supersuit-tech/permission-slip/db"
)

const defaultTestDatabaseURL = "postgres://localhost:5432/permission_slip_test?sslmode=disable"

// TestDatabaseURL returns the database URL for tests.
// It reads from DATABASE_URL_TEST, falling back to a default local URL.
func TestDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL_TEST"); url != "" {
		return url
	}
	return defaultTestDatabaseURL
}

var (
	sharedPool     *pgxpool.Pool
	sharedPoolOnce sync.Once
	sharedPoolErr  error
)

// getSharedPool returns a connection pool shared across all tests in a binary.
// The pool is created once and reused; individual tests get isolated transactions.
func getSharedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	sharedPoolOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		dbURL := TestDatabaseURL()

		// Use a PostgreSQL advisory lock to prevent concurrent migration
		// from multiple test binaries (go test ./... runs packages in parallel).
		sharedPoolErr = migrateWithLock(ctx, dbURL)
		if sharedPoolErr != nil {
			return
		}

		sharedPool, sharedPoolErr = db.Connect(ctx, dbURL)
	})
	if sharedPoolErr != nil {
		t.Fatalf("failed to initialize shared test pool: %v", sharedPoolErr)
	}

	return sharedPool
}

// migrateWithLock runs migrations while holding a PostgreSQL advisory lock.
// This prevents race conditions when multiple test binaries (from go test ./...)
// run migrations concurrently against the same database.
func migrateWithLock(ctx context.Context, dbURL string) error {
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect for migration: %w", err)
	}
	defer pool.Close()

	// Use a single dedicated connection so that lock and unlock run on the same session.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for advisory lock: %w", err)
	}
	defer conn.Release()

	// Acquire an advisory lock (key chosen arbitrarily, must be consistent).
	const lockID = 123456789
	_, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockID) //nolint:errcheck

	return db.Migrate(ctx, dbURL)
}

// SetupTestDB returns a db.DBTX backed by a transaction that is automatically
// rolled back when the test completes. This provides complete data isolation:
// each test's inserts/updates/deletes are invisible to other tests and leave
// no residue in the database.
//
// The underlying connection pool is shared across all tests in a binary,
// so SetupTestDB is cheap to call and safe for use with t.Parallel().
func SetupTestDB(t *testing.T) db.DBTX {
	t.Helper()

	pool := getSharedPool(t)
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("failed to begin test transaction: %v", err)
	}

	t.Cleanup(func() {
		// Rollback discards all changes made during the test.
		tx.Rollback(context.Background()) //nolint:errcheck
	})

	return tx
}

// SetupPool creates a connection pool for testing and runs all migrations.
// Deprecated: Use SetupTestDB for transaction-isolated tests. SetupPool is
// retained for tests that need a raw pool (e.g., tests that manage their own
// transactions or test pool-level behavior).
func SetupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return getSharedPool(t)
}
