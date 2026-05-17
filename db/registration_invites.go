package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	// advisoryLockNSInviteRateLimit is the namespace for per-user advisory
	// locks used in CreateRegistrationInviteIfUnderLimit. Using a named
	// constant avoids accidental collisions with other advisory lock callers.
	advisoryLockNSInviteRateLimit = 1
)

// RegistrationInvite represents a row from the registration_invites table.
type RegistrationInvite struct {
	ID                   string
	UserID               string
	InviteCodeHash       string
	Status               string
	VerificationAttempts int
	ExpiresAt            time.Time
	ConsumedAt           *time.Time
	CreatedAt            time.Time
}

// scanRegistrationInvite scans the standard RETURNING/SELECT column list for registration_invites.
func scanRegistrationInvite(row rowScanner) (*RegistrationInvite, error) {
	var ri RegistrationInvite
	var expiresAt, consumedAt, createdAt sql.NullString
	err := row.Scan(
		&ri.ID, &ri.UserID, &ri.InviteCodeHash, &ri.Status, &ri.VerificationAttempts,
		&expiresAt, &consumedAt, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	var err2 error
	ri.ExpiresAt, err2 = sqliteTimeRequired(expiresAt)
	if err2 != nil {
		return nil, err2
	}
	ri.ConsumedAt, err2 = sqliteTimePtr(consumedAt)
	if err2 != nil {
		return nil, err2
	}
	ri.CreatedAt, err2 = sqliteTimeRequired(createdAt)
	if err2 != nil {
		return nil, err2
	}
	return &ri, nil
}

// CountRecentInvitesByUser returns the number of invites created by the user
// within the given window. Used for per-user rate limiting.
func CountRecentInvitesByUser(ctx context.Context, db DBTX, userID string, window time.Duration) (int, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM registration_invites
		 WHERE user_id = $1 AND created_at > strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-' || $2 || ' seconds')`,
		userID, int(window.Seconds()),
	).Scan(&count)
	return count, err
}

// CreateRegistrationInvite inserts a new registration invite and returns the created row.
// ttlSeconds controls the invite lifetime; expires_at is computed by the database
// as strftime('%Y-%m-%dT%H:%M:%fZ', 'now') + ttlSeconds to avoid clock skew between the app and DB servers.
func CreateRegistrationInvite(ctx context.Context, db DBTX, id, userID, inviteCodeHash string, ttlSeconds int) (*RegistrationInvite, error) {
	return scanRegistrationInvite(db.QueryRow(ctx,
		`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'active', strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+' || $4 || ' seconds'))
		 RETURNING id, user_id, invite_code_hash, status, verification_attempts, expires_at, consumed_at, created_at`,
		id, userID, inviteCodeHash, ttlSeconds,
	))
}

// CreateRegistrationInviteIfUnderLimit atomically checks the per-user invite
// count and inserts only if the user is still under the rate limit. This
// eliminates the TOCTOU race between counting and inserting.
//
// It acquires a per-user advisory lock (pg_advisory_xact_lock) inside a
// transaction so that concurrent requests for the same user are serialized.
// In READ COMMITTED mode each statement after the lock sees the latest
// committed data, so the count check always reflects prior inserts.
//
// Returns the created invite on success. Returns (nil, nil) if the insert was
// skipped because the user has already reached the limit.
func CreateRegistrationInviteIfUnderLimit(
	ctx context.Context,
	d DBTX,
	id, userID, inviteCodeHash string,
	ttlSeconds int,
	rateWindowSeconds int,
	rateLimit int,
) (*RegistrationInvite, error) {
	txDB, owned, err := BeginOrContinue(ctx, d)
	if err != nil {
		return nil, err
	}
	if owned {
		defer RollbackTx(ctx, txDB) //nolint:errcheck // best-effort on failure path
	}

	// Serialize concurrent inserts for the same user. PostgreSQL uses an
	// advisory lock; SQLite has no equivalent, so we skip the lock there
	// (tests and single-writer deployments still get correct counts).
	if err := execPostgreSQLAdvisoryXactLock(ctx, txDB, advisoryLockNSInviteRateLimit, userID); err != nil {
		return nil, fmt.Errorf("advisory lock: %w", err)
	}

	var count int
	if err := txDB.QueryRow(ctx,
		`SELECT COUNT(*) FROM registration_invites
		 WHERE user_id = $1 AND created_at > strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-' || $2 || ' seconds')`,
		userID, rateWindowSeconds,
	).Scan(&count); err != nil {
		return nil, fmt.Errorf("count recent invites: %w", err)
	}

	if count >= rateLimit {
		// Rate limit reached — don't insert.
		if owned {
			_ = CommitTx(ctx, txDB)
		}
		return nil, nil
	}

	var ri *RegistrationInvite
	ri, err = scanRegistrationInvite(txDB.QueryRow(ctx,
		`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'active', strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+' || $4 || ' seconds'))
		 RETURNING id, user_id, invite_code_hash, status, verification_attempts, expires_at, consumed_at, created_at`,
		id, userID, inviteCodeHash, ttlSeconds,
	))
	if err != nil {
		return nil, err
	}

	if owned {
		if err := CommitTx(ctx, txDB); err != nil {
			return nil, fmt.Errorf("commit: %w", err)
		}
	}

	return ri, nil
}

// ConsumeInvite looks up an active invite by its code hash, validates that it is
// not expired, consumed, or locked (>= 5 verification_attempts), and atomically
// marks it as consumed. Returns the invite (including user_id) on success.
//
// The UPDATE ... WHERE uses status='active' AND julianday(expires_at) > julianday('now') AND
// verification_attempts < 5 so that concurrent callers race on the row lock;
// at most one succeeds.
func ConsumeInvite(ctx context.Context, db DBTX, inviteCodeHash string) (*RegistrationInvite, error) {
	ri, err := scanRegistrationInvite(db.QueryRow(ctx,
		`UPDATE registration_invites
		 SET status = 'consumed', consumed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		 WHERE invite_code_hash = $1
		   AND status = 'active'
		   AND julianday(expires_at) > julianday('now')
		   AND verification_attempts < 5
		 RETURNING id, user_id, invite_code_hash, status, verification_attempts, expires_at, consumed_at, created_at`,
		inviteCodeHash,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ri, err
}

// LookupInviteByCodeHash returns an invite by its code hash regardless of status.
// Used to distinguish "not found" from "expired" or "locked" in the API handler.
func LookupInviteByCodeHash(ctx context.Context, db DBTX, inviteCodeHash string) (*RegistrationInvite, error) {
	ri, err := scanRegistrationInvite(db.QueryRow(ctx,
		`SELECT id, user_id, invite_code_hash, status, verification_attempts, expires_at, consumed_at, created_at
		 FROM registration_invites
		 WHERE invite_code_hash = $1`,
		inviteCodeHash,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return ri, err
}
