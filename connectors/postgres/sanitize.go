package postgres

import (
	"fmt"
	"regexp"
	"strings"
)

// identifierPattern matches valid PostgreSQL identifiers:
// letters, digits, underscores, with optional schema prefix.
var identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)

// validateIdentifier checks that name is a safe SQL identifier (table or column).
// It rejects anything that could be used for SQL injection.
func validateIdentifier(name, label string) error {
	if name == "" {
		return fmt.Errorf("%s cannot be empty", label)
	}
	if len(name) > 128 {
		return fmt.Errorf("%s exceeds maximum length of 128 characters", label)
	}
	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("%s %q contains invalid characters (must be alphanumeric/underscore, optionally schema-qualified)", label, name)
	}
	return nil
}

// quoteIdentifier quotes a SQL identifier using double quotes to prevent
// injection. It also escapes any embedded double quotes per SQL standard.
func quoteIdentifier(name string) string {
	// Handle schema.table format
	if idx := strings.IndexByte(name, '.'); idx >= 0 {
		schema := name[:idx]
		table := name[idx+1:]
		return `"` + strings.ReplaceAll(schema, `"`, `""`) + `"."` + strings.ReplaceAll(table, `"`, `""`) + `"`
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// sortedKeys returns the keys of a map in a stable order for deterministic SQL generation.
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Sort for deterministic output.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
