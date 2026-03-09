package supabase

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// parseAndValidate unmarshals JSON parameters and validates them.
func parseAndValidate[T interface{ validate() error }](raw json.RawMessage, dest *T) error {
	if err := json.Unmarshal(raw, dest); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	return (*dest).validate()
}

// validateTable checks the table name is non-empty, contains only safe
// characters, and (if an allowlist is provided) that the table is allowed.
func validateTable(table string, allowedTables []string) error {
	if table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if !isTableNameSafe(table) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid table name %q: must contain only letters, digits, underscores, hyphens, or dots", table),
		}
	}
	if len(allowedTables) > 0 {
		for _, t := range allowedTables {
			if t == table {
				return nil
			}
		}
		return &connectors.ValidationError{
			Message: fmt.Sprintf("table %q is not in the allowed tables list: %v", table, allowedTables),
		}
	}
	return nil
}

// isTableNameSafe checks that a table name contains only characters safe for
// use in PostgREST URL paths: ASCII letters, digits, underscores, hyphens,
// and dots (for schema-qualified names like "public.users").
func isTableNameSafe(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '_' || c == '-' || c == '.') {
			return false
		}
	}
	return true
}

// hasControlChars returns true if s contains ASCII control characters.
func hasControlChars(s string) bool {
	for _, c := range s {
		if c < 0x20 || c == 0x7f {
			return true
		}
	}
	return false
}

// reservedQueryParams lists PostgREST query parameter names that must not
// be used as filter column names. If a filter key matches one of these, it
// would overwrite an explicitly set parameter — potentially bypassing
// pagination limits, changing selected columns, or altering query semantics.
var reservedQueryParams = map[string]bool{
	"select": true, "order": true, "limit": true, "offset": true,
	"on_conflict": true, "columns": true, "or": true, "and": true,
	"not": true,
}

// isColumnNameSafe checks that a column name contains only characters that
// are valid in PostgreSQL column identifiers: ASCII letters, digits,
// underscores. This prevents injection of PostgREST special syntax.
func isColumnNameSafe(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// validFilterOperators lists the PostgREST filter operators that are
// recognized and safe to pass through.
var validFilterOperators = map[string]bool{
	"eq": true, "neq": true, "gt": true, "gte": true,
	"lt": true, "lte": true, "like": true, "ilike": true,
	"is": true, "in": true, "cs": true, "cd": true,
	"sl": true, "sr": true, "nxl": true, "nxr": true,
	"adj": true, "ov": true, "fts": true, "plfts": true,
	"phfts": true, "wfts": true, "not.eq": true, "not.neq": true,
	"not.gt": true, "not.gte": true, "not.lt": true, "not.lte": true,
	"not.like": true, "not.ilike": true, "not.is": true, "not.in": true,
	"not.cs": true, "not.cd": true, "not.sl": true, "not.sr": true,
	"not.nxl": true, "not.nxr": true, "not.adj": true, "not.ov": true,
	"not.fts": true, "not.plfts": true, "not.phfts": true, "not.wfts": true,
}

// validateFilters checks that all filter column names are safe and values
// use valid PostgREST operator prefixes. Returns a helpful error if an
// invalid operator or unsafe column name is found.
func validateFilters(filters map[string]string) error {
	for col, opVal := range filters {
		// Reject column names that collide with PostgREST reserved params.
		if reservedQueryParams[col] {
			return &connectors.ValidationError{
				Message: fmt.Sprintf(
					"filter column name %q is a reserved PostgREST parameter and cannot be used as a filter key",
					col,
				),
			}
		}
		// Validate column name characters.
		if !isColumnNameSafe(col) {
			return &connectors.ValidationError{
				Message: fmt.Sprintf(
					"invalid filter column name %q: must contain only letters, digits, and underscores",
					col,
				),
			}
		}
		dotIdx := strings.IndexByte(opVal, '.')
		if dotIdx < 0 {
			return &connectors.ValidationError{
				Message: fmt.Sprintf(
					"invalid filter for column %q: value %q must use PostgREST operator syntax like 'eq.value', 'gte.18', 'in.(a,b,c)' — see https://postgrest.org/en/stable/references/api/tables_views.html#operators",
					col, opVal,
				),
			}
		}
		op := opVal[:dotIdx]
		// Handle "not.op.value" by extracting "not.op".
		if op == "not" {
			secondDot := strings.IndexByte(opVal[dotIdx+1:], '.')
			if secondDot >= 0 {
				op = opVal[:dotIdx+1+secondDot]
			}
		}
		if !validFilterOperators[op] {
			return &connectors.ValidationError{
				Message: fmt.Sprintf(
					"unknown filter operator %q for column %q: valid operators are eq, neq, gt, gte, lt, lte, like, ilike, is, in, cs, cd, ov (and not.* variants) — see https://postgrest.org/en/stable/references/api/tables_views.html#operators",
					op, col,
				),
			}
		}
	}
	return nil
}

// applyFilters validates and adds PostgREST filter query parameters to the
// URL values. Filters are key-value pairs where the key is the column name
// and the value is an "operator.value" string (e.g., "eq.active", "gte.18").
func applyFilters(q url.Values, filters map[string]string) error {
	if err := validateFilters(filters); err != nil {
		return err
	}
	for col, opVal := range filters {
		q.Set(col, opVal)
	}
	return nil
}

// returningSelect sets the PostgREST select param for returned rows,
// defaulting to "*" if not specified.
func returningSelect(q url.Values, returning string) {
	if returning == "" {
		returning = "*"
	}
	q.Set("select", returning)
}
