package db

import (
	"context"
	"fmt"
)

// Advisory lock namespaces for plan limit checks. Each namespace ensures
// per-user serialization of count+insert to prevent TOCTOU races where
// concurrent requests could both pass the limit check and exceed the plan cap.
//
// The two-argument form of pg_advisory_xact_lock(ns, hashtext(user_id))
// scopes each lock to one (namespace, user) pair, so different resource
// types don't block each other.
const (
	advisoryLockNSAgentLimit           = 10
	advisoryLockNSStandingApprovalLimit = 11
	advisoryLockNSCredentialLimit      = 12
)

// AcquirePlanLimitLock acquires a transaction-scoped advisory lock for the
// given namespace and user. This serializes concurrent requests so that
// count+insert is atomic, preventing TOCTOU races on plan limit checks.
//
// Must be called within a transaction — the lock is released when the
// transaction commits or rolls back.
func AcquirePlanLimitLock(ctx context.Context, tx DBTX, namespace int, userID string) error {
	_, err := tx.Exec(ctx,
		`SELECT pg_advisory_xact_lock($1, hashtext($2))`,
		namespace, userID)
	if err != nil {
		return fmt.Errorf("advisory lock (ns=%d): %w", namespace, err)
	}
	return nil
}

// AcquireAgentLimitLock acquires a per-user advisory lock for agent limit checks.
func AcquireAgentLimitLock(ctx context.Context, tx DBTX, userID string) error {
	return AcquirePlanLimitLock(ctx, tx, advisoryLockNSAgentLimit, userID)
}

// AcquireStandingApprovalLimitLock acquires a per-user advisory lock for standing approval limit checks.
func AcquireStandingApprovalLimitLock(ctx context.Context, tx DBTX, userID string) error {
	return AcquirePlanLimitLock(ctx, tx, advisoryLockNSStandingApprovalLimit, userID)
}

// AcquireCredentialLimitLock acquires a per-user advisory lock for credential limit checks.
func AcquireCredentialLimitLock(ctx context.Context, tx DBTX, userID string) error {
	return AcquirePlanLimitLock(ctx, tx, advisoryLockNSCredentialLimit, userID)
}
