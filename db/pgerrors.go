package db

// Legacy error helpers retained for compatibility with call sites mid-migration.
// New code should call db.IsUniqueViolation / db.IsForeignKeyViolation directly
// (defined in db.go), which match SQLite errors via stable message substrings.

const (
	// PgCodeForeignKeyViolation kept as a deprecated alias; matches nothing
	// now that we're on SQLite. Use IsForeignKeyViolation instead.
	//
	// Deprecated: use IsForeignKeyViolation.
	PgCodeForeignKeyViolation = "23503"
)

// isUniqueViolation is the package-private helper. Delegates to the exported
// IsUniqueViolation so we keep one source of truth for the SQLite check.
func isUniqueViolation(err error) bool {
	return IsUniqueViolation(err)
}
