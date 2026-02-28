package db

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL error codes used across the application.
// See: https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	// PgCodeUniqueViolation is raised when an INSERT or UPDATE violates
	// a UNIQUE constraint (Class 23 — Integrity Constraint Violation).
	PgCodeUniqueViolation = "23505"

	// PgCodeForeignKeyViolation is raised when an INSERT or UPDATE violates
	// a FOREIGN KEY constraint (Class 23 — Integrity Constraint Violation).
	PgCodeForeignKeyViolation = "23503"
)

// isUniqueViolation returns true if err is a PostgreSQL unique violation.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == PgCodeUniqueViolation
}
