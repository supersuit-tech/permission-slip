package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OnboardingError represents a typed error from CreateProfile.
type OnboardingError struct {
	Code    string
	Message string
}

func (e *OnboardingError) Error() string { return e.Message }

const (
	OnboardingErrUsernameTaken  = "username_taken"
	OnboardingErrProfileExists = "profile_exists"
)

// CreateProfile provisions a new user: inserts an auth.users row (if not
// present — Supabase manages this in production; local Postgres needs it
// inserted manually) and a profiles row.
//
// Returns the created Profile, or an *OnboardingError if the username is
// already taken (OnboardingErrUsernameTaken) or if a profile already exists
// for this user due to a concurrent request (OnboardingErrProfileExists).
func CreateProfile(ctx context.Context, db DBTX, userID, username string, marketingOptIn bool) (*Profile, error) {
	// Upsert auth.users so the FK from profiles is satisfied.
	// In production (Supabase), auth.users is managed by Supabase and this
	// row already exists by the time the user reaches onboarding.
	// In local dev (standalone Postgres), we insert it ourselves.
	_, err := db.Exec(ctx,
		`INSERT INTO auth.users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`,
		userID,
	)
	if err != nil {
		return nil, err
	}

	var p Profile
	err = db.QueryRow(ctx,
		`INSERT INTO profiles (id, username, marketing_opt_in) VALUES ($1, $2, $3)
		 RETURNING id, username, marketing_opt_in, created_at`,
		userID, username, marketingOptIn,
	).Scan(&p.ID, &p.Username, &p.MarketingOptIn, &p.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgCodeUniqueViolation {
			// Distinguish username uniqueness (profiles_username_key) from
			// PK conflict (profiles_pkey) to avoid misleading error messages
			// on concurrent double-submits for the same user.
			if pgErr.ConstraintName == "profiles_username_key" {
				return nil, &OnboardingError{
					Code:    OnboardingErrUsernameTaken,
					Message: "username is already taken",
				}
			}
			// PK conflict: profile already exists for this user (concurrent request).
			return nil, &OnboardingError{
				Code:    OnboardingErrProfileExists,
				Message: "profile already exists",
			}
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	p.CreatedAt = p.CreatedAt.UTC().Truncate(time.Millisecond)
	return &p, nil
}
