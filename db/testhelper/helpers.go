package testhelper

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

// mustExec runs a SQL statement and fails the test if it errors.
// Use this in fixture functions to reduce boilerplate.
func mustExec(t *testing.T, d db.DBTX, sql string, args ...any) {
	t.Helper()
	if _, err := d.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("mustExec: %v\nSQL: %s", err, sql)
	}
}

// MustExec is an exported version of mustExec for use in external test packages.
func MustExec(t *testing.T, d db.DBTX, sql string, args ...any) {
	t.Helper()
	mustExec(t, d, sql, args...)
}

// WithSavepoint wraps fn in a PostgreSQL savepoint so that a failed SQL
// statement (e.g. a CHECK-constraint violation) does not abort the enclosing
// transaction. It returns whatever error fn returned.
//
// Use this in tests that deliberately trigger constraint errors:
//
//	err := testhelper.WithSavepoint(t, tx, func() error {
//	    _, err := tx.Exec(ctx, "INSERT INTO ... invalid ...")
//	    return err
//	})
//	if err == nil { t.Error("expected constraint error") }
func WithSavepoint(t *testing.T, d db.DBTX, fn func() error) error {
	t.Helper()
	ctx := context.Background()

	sp := "sp_" + hex.EncodeToString(mustRandBytes(t, 4))
	_, err := d.Exec(ctx, "SAVEPOINT "+sp)
	if err != nil {
		t.Fatalf("WithSavepoint: create savepoint: %v", err)
	}

	fnErr := fn()

	if fnErr != nil {
		// Statement failed — roll back to savepoint to restore the transaction.
		_, err = d.Exec(ctx, "ROLLBACK TO SAVEPOINT "+sp)
	} else {
		// Statement succeeded — release the savepoint (no-op on data).
		_, err = d.Exec(ctx, "RELEASE SAVEPOINT "+sp)
	}
	if err != nil {
		t.Fatalf("WithSavepoint: cleanup savepoint: %v", err)
	}

	return fnErr
}

// mustRandBytes returns n random bytes, failing the test on error.
func mustRandBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("failed to generate random bytes: %v", err)
	}
	return b
}

// GenerateUID generates a random UUID v4 for use in tests.
// Each test should call this to get a unique user ID, avoiding
// cross-test and cross-package collisions.
func GenerateUID(t *testing.T) string {
	t.Helper()
	b := mustRandBytes(t, 16)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 1
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// GenerateID generates a random string ID with the given prefix for use in tests.
func GenerateID(t *testing.T, prefix string) string {
	t.Helper()
	return prefix + hex.EncodeToString(mustRandBytes(t, 8))
}
