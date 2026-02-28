package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ConsumedTokenError represents a domain error from consumed-token operations.
type ConsumedTokenError struct {
	Code string
}

func (e *ConsumedTokenError) Error() string { return e.Code }

const (
	// ConsumedTokenErrAlreadyConsumed is returned when a token JTI has
	// already been inserted into consumed_tokens (replay attempt).
	ConsumedTokenErrAlreadyConsumed = "already_consumed"
)

// ConsumeToken atomically inserts a JTI into consumed_tokens. If the JTI
// already exists (unique constraint on the PK), it returns a
// ConsumedTokenError with code already_consumed. This is the single-use
// enforcement mechanism for action tokens.
func ConsumeToken(ctx context.Context, db DBTX, jti string) error {
	_, err := db.Exec(ctx,
		`INSERT INTO consumed_tokens (jti) VALUES ($1)`,
		jti,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return &ConsumedTokenError{Code: ConsumedTokenErrAlreadyConsumed}
		}
		return err
	}
	return nil
}

// IsTokenConsumed checks whether a JTI already exists in consumed_tokens.
func IsTokenConsumed(ctx context.Context, db DBTX, jti string) (bool, error) {
	var consumedAt time.Time
	err := db.QueryRow(ctx,
		`SELECT consumed_at FROM consumed_tokens WHERE jti = $1`,
		jti,
	).Scan(&consumedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
