package postgres

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/pkg/sqldb"
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

// validateWhereCols validates all column names in a WHERE map.
func validateWhereCols(where map[string]interface{}) error {
	for col := range where {
		if err := validateIdentifier(col, "where column"); err != nil {
			return &connectors.ValidationError{Message: err.Error()}
		}
	}
	return nil
}

// validateReturningCols validates all column names in a RETURNING list.
func validateReturningCols(cols []string) error {
	for _, col := range cols {
		if col != "*" {
			if err := validateIdentifier(col, "returning column"); err != nil {
				return &connectors.ValidationError{Message: err.Error()}
			}
		}
	}
	return nil
}

// buildWhereClause builds a parameterized WHERE clause from a column-value map.
// NULL values produce IS NULL conditions; non-null values use $N placeholders.
// startIdx is the first placeholder index to use.
func buildWhereClause(where map[string]interface{}, startIdx int) (string, []interface{}) {
	cols := sqldb.SortedKeys(where)
	var clauses []string
	var args []interface{}
	paramIdx := startIdx

	for _, col := range cols {
		val := where[col]
		if val == nil {
			clauses = append(clauses, fmt.Sprintf("%s IS NULL", quoteIdentifier(col)))
		} else {
			clauses = append(clauses, fmt.Sprintf("%s = $%d", quoteIdentifier(col), paramIdx))
			args = append(args, val)
			paramIdx++
		}
	}

	return strings.Join(clauses, " AND "), args
}
