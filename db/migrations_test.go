package db_test

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// TestMigrationVersionsUnique ensures no two migration files share the same
// goose version (the numeric prefix before the first underscore). Duplicate
// versions cause goose to panic at startup when it sorts the migration list.
//
// This test uses the embedded filesystem so it requires no database — it runs
// in CI as a fast, zero-dependency guard against accidental timestamp collisions.
func TestMigrationVersionsUnique(t *testing.T) {
	t.Parallel()

	seen := make(map[string]string) // version -> filename

	err := fs.WalkDir(db.Migrations, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".sql" {
			return nil
		}

		name := d.Name()
		version := extractVersion(name)
		if version == "" {
			t.Errorf("migration %q has no numeric version prefix", name)
			return nil
		}

		if existing, ok := seen[version]; ok {
			t.Errorf("duplicate migration version %s:\n  %s\n  %s", version, existing, name)
		} else {
			seen[version] = name
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk migrations directory: %v", err)
	}

	if len(seen) == 0 {
		t.Fatal("no migration files found — embed may be broken")
	}

	t.Logf("verified %d migrations have unique versions", len(seen))
}

// TestMigrationVersionsOrdered verifies that migration files are ordered by
// their numeric version prefix. Out-of-order migrations can cause subtle issues
// with goose when a migration sorts before an already-applied one.
func TestMigrationVersionsOrdered(t *testing.T) {
	t.Parallel()

	var versions []string
	var names []string

	err := fs.WalkDir(db.Migrations, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".sql" {
			return nil
		}

		name := d.Name()
		version := extractVersion(name)
		if version == "" {
			return nil
		}

		versions = append(versions, version)
		names = append(names, name)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk migrations directory: %v", err)
	}

	// fs.WalkDir returns entries in lexical order, which for our naming
	// scheme (numeric prefix) should also be version order.
	for i := 1; i < len(versions); i++ {
		if versions[i] < versions[i-1] {
			t.Errorf("migration out of order:\n  %s (version %s)\n  comes after\n  %s (version %s)",
				names[i], versions[i], names[i-1], versions[i-1])
		}
	}
}

// extractVersion returns the numeric prefix of a migration filename.
// For "20260228300003_data_retention_compliance.sql" it returns "20260228300003".
// For "00001_init.sql" it returns "00001".
func extractVersion(filename string) string {
	// goose expects the version to be the leading digits before the first underscore
	idx := strings.Index(filename, "_")
	if idx <= 0 {
		return ""
	}
	prefix := filename[:idx]
	// Verify it's all digits
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return prefix
}

// TestExtractVersion validates the helper function.
func TestExtractVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"20260228300003_data_retention_compliance.sql", "20260228300003"},
		{"00001_init.sql", "00001"},
		{"no_version.sql", ""},
		{"_leading_underscore.sql", ""},
		{"abc_not_numeric.sql", ""},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := extractVersion(tc.input)
			if got != tc.want {
				t.Errorf("extractVersion(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

