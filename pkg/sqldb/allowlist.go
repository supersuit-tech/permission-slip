package sqldb

import (
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// CheckTableAllowed validates that a table name is in the allowlist.
// If allowedTables is empty, all tables are permitted.
func CheckTableAllowed(table string, allowedTables []string) error {
	if len(allowedTables) == 0 {
		return nil
	}
	for _, t := range allowedTables {
		if strings.EqualFold(t, table) {
			return nil
		}
	}
	return &connectors.ValidationError{
		Message: fmt.Sprintf("table %q is not in the allowed tables list", table),
	}
}

// CheckColumnsAllowed validates that all columns are in the allowlist.
// If allowedColumns is empty, all columns are permitted.
func CheckColumnsAllowed(columns []string, allowedColumns []string) error {
	if len(allowedColumns) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(allowedColumns))
	for _, c := range allowedColumns {
		allowed[strings.ToLower(c)] = true
	}
	for _, c := range columns {
		if !allowed[strings.ToLower(c)] {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("column %q is not in the allowed columns list", c),
			}
		}
	}
	return nil
}
