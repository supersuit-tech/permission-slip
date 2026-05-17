package db

import (
	"context"
	"strings"
)

// execPostgreSQLAdvisoryXactLock runs PostgreSQL's pg_advisory_xact_lock for
// per-user serialization inside a transaction. On SQLite (and other engines
// without these functions), it is a no-op so count+insert flows still work in
// tests and single-writer deployments.
func execPostgreSQLAdvisoryXactLock(ctx context.Context, tx DBTX, namespace int, userID string) error {
	_, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock($1, hashtext($2))`,
		namespace, userID)
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "no such function") &&
		(strings.Contains(msg, "pg_advisory") || strings.Contains(msg, "hashtext")) {
		return nil
	}
	return err
}
