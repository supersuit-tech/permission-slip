package testhelper

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// safeIdentifier matches valid SQL identifiers (lowercase letters, digits, underscores, dots).
// Used to validate table names and simple WHERE clauses before interpolation.
var safeIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_.]+$`)

// TableColumns returns the set of column names for a table in the public schema.
func TableColumns(t *testing.T, d db.DBTX, table string) map[string]bool {
	t.Helper()
	rows, err := d.Query(context.Background(),
		"SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1",
		table)
	if err != nil {
		t.Fatalf("failed to query columns for %s: %v", table, err)
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan column name: %v", err)
		}
		cols[name] = true
	}
	if len(cols) == 0 {
		t.Fatalf("table %s has no columns (does it exist?)", table)
	}
	return cols
}

// RequireColumns asserts that every column in want exists in the given table.
func RequireColumns(t *testing.T, d db.DBTX, table string, want []string) {
	t.Helper()
	cols := TableColumns(t, d, table)
	for _, c := range want {
		if !cols[c] {
			t.Errorf("table %s: expected column %q not found (have %v)", table, c, cols)
		}
	}
}

// RequireCascadeDeletes runs deleteSQL and then asserts that zero rows match
// whereClause in each of the child tables. Call this after inserting parent
// and child rows.
//
// Table names are validated against a safe identifier pattern before
// interpolation to prevent accidental SQL injection.
func RequireCascadeDeletes(t *testing.T, d db.DBTX, deleteSQL string, childTables []string, whereClause string) {
	t.Helper()
	ctx := context.Background()

	for _, table := range childTables {
		if !safeIdentifier.MatchString(table) {
			t.Fatalf("RequireCascadeDeletes: unsafe table name %q", table)
		}
	}

	_, err := d.Exec(ctx, deleteSQL)
	if err != nil {
		t.Fatalf("cascade delete failed: %v", err)
	}

	for _, table := range childTables {
		var count int
		err := d.QueryRow(ctx,
			fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, whereClause),
		).Scan(&count)
		if err != nil {
			t.Fatalf("failed to count rows in %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("expected 0 rows in %s after cascade delete, got %d", table, count)
		}
	}
}

// RequireIndex asserts that an index with the given name exists on the
// specified table in the public schema.
func RequireIndex(t *testing.T, d db.DBTX, table, indexName string) {
	t.Helper()
	var exists bool
	err := d.QueryRow(context.Background(), `
		SELECT EXISTS(
			SELECT 1 FROM pg_indexes
			WHERE schemaname = 'public'
			  AND tablename = $1
			  AND indexname = $2
		)`, table, indexName).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check index %s on %s: %v", indexName, table, err)
	}
	if !exists {
		t.Errorf("expected index %s on table %s, but it does not exist", indexName, table)
	}
}

// RequireCheckValues validates that a CHECK constraint accepts validValues and
// rejects invalidValue. The insertFn callback should attempt to insert a row
// using the given value; the index parameter provides a unique suffix for
// generating distinct primary keys across iterations.
//
// Invalid-value insertions are wrapped in savepoints so that constraint
// violations don't abort the enclosing transaction.
func RequireCheckValues(t *testing.T, d db.DBTX, description string, validValues []string, invalidValue string, insertFn func(value string, index int) error) {
	t.Helper()
	for i, val := range validValues {
		if err := insertFn(val, i); err != nil {
			t.Errorf("valid %s %q was rejected: %v", description, val, err)
		}
	}
	err := WithSavepoint(t, d, func() error {
		return insertFn(invalidValue, len(validValues))
	})
	if err == nil {
		t.Errorf("expected CHECK constraint violation for invalid %s %q, but insert succeeded", description, invalidValue)
	}
}

// RequireUniqueViolation asserts that the first insert succeeds and the
// duplicate insert is rejected by a UNIQUE constraint.
//
// The duplicate insertion is wrapped in a savepoint so that the constraint
// violation doesn't abort the enclosing transaction.
func RequireUniqueViolation(t *testing.T, d db.DBTX, description string, firstInsert, duplicateInsert func() error) {
	t.Helper()
	if err := firstInsert(); err != nil {
		t.Fatalf("first insert for %s uniqueness test failed: %v", description, err)
	}
	err := WithSavepoint(t, d, func() error {
		return duplicateInsert()
	})
	if err == nil {
		t.Errorf("expected UNIQUE violation for %s, but insert succeeded", description)
	}
}

// RequireConstraintExists asserts that a constraint with the given name
// and type exists on the specified table. constraintType should be one of:
// "UNIQUE", "CHECK", "PRIMARY KEY", "FOREIGN KEY".
func RequireConstraintExists(t *testing.T, d db.DBTX, table, constraintName, constraintType string) {
	t.Helper()
	var exists bool
	err := d.QueryRow(context.Background(), `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.table_constraints
			WHERE table_schema = 'public'
			  AND table_name = $1
			  AND constraint_name = $2
			  AND constraint_type = $3
		)`, table, constraintName, constraintType).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check constraint %s on %s: %v", constraintName, table, err)
	}
	if !exists {
		t.Errorf("expected %s constraint %q on table %s, but it does not exist", constraintType, constraintName, table)
	}
}

// RequireRowValue asserts that a single row identified by idCol=idVal
// has the expected value in the given column.
//
// Table and column names are validated against a safe identifier pattern
// before interpolation.
func RequireRowValue(t *testing.T, d db.DBTX, table, idCol, idVal, col, want string) {
	t.Helper()
	for _, name := range []string{table, idCol, col} {
		if !safeIdentifier.MatchString(name) {
			t.Fatalf("RequireRowValue: unsafe identifier %q", name)
		}
	}
	var got string
	err := d.QueryRow(context.Background(),
		fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", col, table, idCol),
		idVal,
	).Scan(&got)
	if err != nil {
		t.Fatalf("RequireRowValue: failed to query %s.%s where %s=%q: %v", table, col, idCol, idVal, err)
	}
	if got != want {
		t.Errorf("expected %s.%s = %q where %s=%q, got %q", table, col, want, idCol, idVal, got)
	}
}

// RequireAuditEventCount asserts that exactly `want` audit events exist
// matching the given userID and eventType.
func RequireAuditEventCount(t *testing.T, d db.DBTX, userID, eventType string, want int) {
	t.Helper()
	var count int
	err := d.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_events WHERE user_id = $1 AND event_type = $2`,
		userID, eventType,
	).Scan(&count)
	if err != nil {
		t.Fatalf("RequireAuditEventCount: failed to query audit_events: %v", err)
	}
	if count != want {
		t.Errorf("expected %d %s audit event(s) for user %s, got %d", want, eventType, userID, count)
	}
}

// RequirePgCronJob asserts that a pg_cron job with the given name is
// scheduled. Skips the test if pg_cron is not available.
func RequirePgCronJob(t *testing.T, d db.DBTX, jobName string) {
	t.Helper()
	ctx := context.Background()

	var hasCronSchema bool
	err := d.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'cron')`).Scan(&hasCronSchema)
	if err != nil {
		t.Fatalf("failed to check for cron schema: %v", err)
	}
	if !hasCronSchema {
		t.Skip("pg_cron not available, skipping cron job verification")
	}

	var exists bool
	err = d.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM cron.job WHERE jobname = $1)`, jobName).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check cron job %s: %v", jobName, err)
	}
	if !exists {
		t.Errorf("expected pg_cron job %q to be scheduled, but it was not found", jobName)
	}
}
