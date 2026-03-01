package db_test

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// TestMigrationTimestampsUnique ensures no two migration files share the same
// numeric timestamp prefix. Duplicate timestamps cause non-deterministic
// migration ordering and runtime failures.
func TestMigrationTimestampsUnique(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(db.Migrations, "migrations")
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	seen := make(map[string]string) // timestamp -> filename
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Extract the numeric prefix before the first underscore.
		idx := strings.Index(name, "_")
		if idx <= 0 {
			continue // skip files without a timestamp prefix (e.g. README)
		}
		timestamp := name[:idx]

		if prev, exists := seen[timestamp]; exists {
			t.Errorf("duplicate migration timestamp %s:\n  %s\n  %s", timestamp, prev, name)
		}
		seen[timestamp] = name
	}

	if len(seen) == 0 {
		t.Fatal("no migration files found — check embed path")
	}
}

// TestMigrationFilesOrdered ensures migration files are in chronological order
// by their timestamp prefixes. Out-of-order migrations can cause subtle
// dependency issues.
func TestMigrationFilesOrdered(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(db.Migrations, "migrations")
	if err != nil {
		t.Fatalf("failed to read migrations directory: %v", err)
	}

	var prevTimestamp, prevName string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		idx := strings.Index(name, "_")
		if idx <= 0 {
			continue
		}
		timestamp := name[:idx]

		if prevTimestamp != "" && timestamp < prevTimestamp {
			t.Errorf("migration files out of order:\n  %s (timestamp %s)\n  comes after %s (timestamp %s)",
				name, timestamp, prevName, prevTimestamp)
		}
		prevTimestamp = timestamp
		prevName = name
	}
}
