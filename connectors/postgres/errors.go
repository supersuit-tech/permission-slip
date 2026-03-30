package postgres

import (
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// mapPgError maps PostgreSQL errors to typed connector errors.
func mapPgError(err error, action string) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	// Check for context/timeout errors first.
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("PostgreSQL %s timed out: %v", action, err)}
	}

	// pgx error codes are embedded in the error message.
	// SQLSTATE 28000 = invalid_authorization_specification
	// SQLSTATE 28P01 = invalid_password
	if containsAny(msg, "28000", "28P01", "password authentication failed", "no pg_hba.conf entry") {
		return &connectors.AuthError{Message: fmt.Sprintf("PostgreSQL authentication failed: %v", err)}
	}

	// Connection refused / unreachable.
	if containsAny(msg, "connection refused", "no such host", "could not connect", "connection reset") {
		return &connectors.ExternalError{Message: fmt.Sprintf("PostgreSQL connection failed: %v", err)}
	}

	// Permission denied (SQLSTATE 42501).
	if strings.Contains(msg, "42501") || strings.Contains(msg, "permission denied") {
		return &connectors.AuthError{Message: fmt.Sprintf("PostgreSQL permission denied: %v", err)}
	}

	// Syntax error or invalid SQL (SQLSTATE 42xxx).
	if strings.Contains(msg, "42601") || strings.Contains(msg, "syntax error") {
		return &connectors.ValidationError{Message: fmt.Sprintf("PostgreSQL SQL syntax error: %v", err)}
	}

	// Undefined table/column (SQLSTATE 42P01, 42703).
	if containsAny(msg, "42P01", "42703", "does not exist") {
		return &connectors.ValidationError{Message: fmt.Sprintf("PostgreSQL object not found: %v", err)}
	}

	// Constraint violations (SQLSTATE 23xxx).
	// pgx formats these as "ERROR: ... (SQLSTATE 23NNN)" so we check for that pattern.
	if containsAny(msg, "SQLSTATE 23", "violates unique constraint", "violates not-null constraint",
		"violates foreign key constraint", "violates check constraint", "duplicate key value") {
		return &connectors.ValidationError{Message: fmt.Sprintf("PostgreSQL constraint violation: %v", err)}
	}

	// Statement timeout (SQLSTATE 57014).
	if strings.Contains(msg, "57014") || strings.Contains(msg, "canceling statement due to statement timeout") {
		return &connectors.TimeoutError{Message: fmt.Sprintf("PostgreSQL statement timed out: %v", err)}
	}

	// Read-only transaction violation (SQLSTATE 25006).
	if strings.Contains(msg, "25006") || (strings.Contains(msg, "cannot execute") && strings.Contains(msg, "read-only")) {
		return &connectors.ValidationError{Message: fmt.Sprintf("PostgreSQL read-only transaction violation: %v", err)}
	}

	return &connectors.ExternalError{Message: fmt.Sprintf("PostgreSQL %s failed: %v", action, err)}
}

// containsAny returns true if s contains any of the substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
