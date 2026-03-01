package db_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// TestMigrationTimestampsUnique ensures no two migration files share the same
// timestamp prefix. Goose uses the numeric prefix for ordering; duplicates
// cause non-deterministic application order and potential failures.
func TestMigrationTimestampsUnique(t *testing.T) {
	t.Parallel()

	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	seen := make(map[string]string) // timestamp -> filename
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		ts := extractTimestamp(name)
		if ts == "" {
			t.Errorf("migration file %q has no numeric timestamp prefix", name)
			continue
		}

		if prev, ok := seen[ts]; ok {
			t.Errorf("duplicate migration timestamp %s:\n  %s\n  %s", ts, prev, name)
		}
		seen[ts] = name
	}
}

// TestMigrationTimestampsSorted ensures migration files are in strictly
// ascending order by their timestamp prefix. Out-of-order migrations can
// cause goose to skip files or apply them in the wrong sequence.
func TestMigrationTimestampsSorted(t *testing.T) {
	t.Parallel()

	dir := migrationsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	var timestamps []struct {
		ts   string
		name string
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		ts := extractTimestamp(name)
		if ts == "" {
			continue
		}
		timestamps = append(timestamps, struct {
			ts   string
			name string
		}{ts, name})
	}

	sorted := sort.SliceIsSorted(timestamps, func(i, j int) bool {
		return timestamps[i].ts < timestamps[j].ts
	})
	if !sorted {
		for i := 1; i < len(timestamps); i++ {
			if timestamps[i].ts < timestamps[i-1].ts {
				t.Errorf("migration out of order: %s (after %s)",
					timestamps[i].name, timestamps[i-1].name)
			}
		}
	}
}

// extractTimestamp returns the leading numeric prefix of a migration filename.
// For "20260216002736_profiles.sql" it returns "20260216002736".
// Returns "" if the filename doesn't start with digits.
func extractTimestamp(filename string) string {
	idx := strings.IndexByte(filename, '_')
	if idx <= 0 {
		return ""
	}
	prefix := filename[:idx]
	for _, c := range prefix {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return prefix
}

// migrationsDir returns the absolute path to the db/migrations directory,
// relative to this test file's location.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test file path")
	}
	return filepath.Join(filepath.Dir(thisFile), "migrations")
}
