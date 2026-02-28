package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

// CountRecentInvitesByUser returns the number of invites created by the user
// within the given window. Used for per-user rate limiting.
func CountRecentInvitesByUser(ctx context.Context, db DBTX, userID string, window time.Duration) (int, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM registration_invites
		 WHERE user_id = $1 AND created_at > now() - make_interval(secs => $2)`,
		userID, int(window.Seconds()),
	).Scan(&count)
	return count, err
}

// CreateRegistrationInvite inserts a new registration invite and returns the created row.
// ttlSeconds controls the invite lifetime; expires_at is computed by the database
// as now() + ttlSeconds to avoid clock skew between the app and DB servers.
func CreateRegistrationInvite(ctx context.Context, db DBTX, id, userID, inviteCodeHash string, ttlSeconds int) (*RegistrationInvite, error) {
	var ri RegistrationInvite
	err := db.QueryRow(ctx,
		`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'active', now() + make_interval(secs => $4))
		 RETURNING id, user_id, invite_code_hash, status, verification_attempts, expires_at, consumed_at, created_at`,
		id, userID, inviteCodeHash, ttlSeconds,
	).Scan(&ri.ID, &ri.UserID, &ri.InviteCodeHash, &ri.Status, &ri.VerificationAttempts, &ri.ExpiresAt, &ri.ConsumedAt, &ri.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &ri, nil
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

	// Serialize concurrent inserts for the same user. The two-argument form
	// scopes the lock to our namespace so it cannot collide with other
	// advisory lock callers. hashtext produces a stable int4 from the user ID.
	if _, err := txDB.Exec(ctx,
		`SELECT pg_advisory_xact_lock($1, hashtext($2))`,
		advisoryLockNSInviteRateLimit, userID); err != nil {
		return nil, fmt.Errorf("advisory lock: %w", err)
	}

	var count int
	if err := txDB.QueryRow(ctx,
		`SELECT COUNT(*) FROM registration_invites
		 WHERE user_id = $1 AND created_at > now() - make_interval(secs => $2)`,
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

	var ri RegistrationInvite
	if err := txDB.QueryRow(ctx,
		`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'active', now() + make_interval(secs => $4))
		 RETURNING id, user_id, invite_code_hash, status, verification_attempts, expires_at, consumed_at, created_at`,
		id, userID, inviteCodeHash, ttlSeconds,
	).Scan(&ri.ID, &ri.UserID, &ri.InviteCodeHash, &ri.Status, &ri.VerificationAttempts, &ri.ExpiresAt, &ri.ConsumedAt, &ri.CreatedAt); err != nil {
		return nil, err
	}

	if owned {
		if err := CommitTx(ctx, txDB); err != nil {
			return nil, fmt.Errorf("commit: %w", err)
		}
	}

	return &ri, nil
}

// ConsumeInvite looks up an active invite by its code hash, validates that it is
// not expired, consumed, or locked (>= 5 verification_attempts), and atomically
// marks it as consumed. Returns the invite (including user_id) on success.
//
// The UPDATE ... WHERE uses status='active' AND expires_at > now() AND
// verification_attempts < 5 so that concurrent callers race on the row lock;
// at most one succeeds.
func ConsumeInvite(ctx context.Context, db DBTX, inviteCodeHash string) (*RegistrationInvite, error) {
	var ri RegistrationInvite
	err := db.QueryRow(ctx,
		`UPDATE registration_invites
		 SET status = 'consumed', consumed_at = now()
		 WHERE invite_code_hash = $1
		   AND status = 'active'
		   AND expires_at > now()
		   AND verification_attempts < 5
		 RETURNING id, user_id, invite_code_hash, status, verification_attempts, expires_at, consumed_at, created_at`,
		inviteCodeHash,
	).Scan(&ri.ID, &ri.UserID, &ri.InviteCodeHash, &ri.Status, &ri.VerificationAttempts, &ri.ExpiresAt, &ri.ConsumedAt, &ri.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ri, nil
}

// LookupInviteByCodeHash returns an invite by its code hash regardless of status.
// Used to distinguish "not found" from "expired" or "locked" in the API handler.
func LookupInviteByCodeHash(ctx context.Context, db DBTX, inviteCodeHash string) (*RegistrationInvite, error) {
	var ri RegistrationInvite
	err := db.QueryRow(ctx,
		`SELECT id, user_id, invite_code_hash, status, verification_attempts, expires_at, consumed_at, created_at
		 FROM registration_invites
		 WHERE invite_code_hash = $1`,
		inviteCodeHash,
	).Scan(&ri.ID, &ri.UserID, &ri.InviteCodeHash, &ri.Status, &ri.VerificationAttempts, &ri.ExpiresAt, &ri.ConsumedAt, &ri.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ri, nil
}
