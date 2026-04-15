package db

import (
	"context"
	"time"
)

// ConsumeSignature records that an agent-signed request with the given signature
// hash has been processed, preventing replays of the same signature. If the hash
// already exists in the table, inserted=false is returned and the caller MUST
// reject the request as a replay.
//
// signatureHash is the SHA-256 of the raw signature bytes. agentID is stored
// for observability — use 0 for pre-registration requests where the agent does
// not yet have an ID. expiresAt is when the row becomes safe to delete; callers
// should set it to signed_timestamp + signature_window + skew_buffer.
func ConsumeSignature(ctx context.Context, db DBTX, signatureHash []byte, agentID int64, expiresAt time.Time) (inserted bool, err error) {
	tag, err := db.Exec(ctx, `
		INSERT INTO consumed_signatures (signature_hash, agent_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (signature_hash) DO NOTHING`,
		signatureHash, agentID, expiresAt,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// CleanupExpiredConsumedSignatures deletes rows whose expires_at has passed.
// The app runs this on a background ticker; pg_cron runs the same DELETE on
// its own schedule (see 20260415134017_add_consumed_signatures.sql). Either
// is sufficient — running both keeps the table bounded even when pg_cron is
// disabled in a given deployment.
//
// Returns the number of rows deleted.
func CleanupExpiredConsumedSignatures(ctx context.Context, db DBTX) (int64, error) {
	tag, err := db.Exec(ctx, `DELETE FROM consumed_signatures WHERE expires_at < now()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
