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
	//
	// In production the app_backend role only has SELECT on auth.users
	// (INSERT is intentionally omitted because Supabase creates the row
	// during OTP login). PostgreSQL checks INSERT privilege before evaluating
	// ON CONFLICT DO NOTHING, so the no-op INSERT raises 42501
	// (insufficient_privilege). We treat this as success: if we lack INSERT
	// permission, the row must already exist (managed by Supabase).
	_, err := db.Exec(ctx,
		`INSERT INTO auth.users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`,
		userID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42501" {
			// insufficient_privilege — expected in production where
			// app_backend lacks INSERT on auth.users. Safe to ignore
			// because Supabase already created the row during login.
		} else {
			return nil, err
		}
	}

	var p Profile
	err = db.QueryRow(ctx,
		`INSERT INTO profiles (id, username, email, marketing_opt_in)
		 VALUES ($1, $2, (SELECT email FROM auth.users WHERE id = $1), $3)
		 RETURNING id, username, email, phone, marketing_opt_in, created_at`,
		userID, username, marketingOptIn,
	).Scan(&p.ID, &p.Username, &p.Email, &p.Phone, &p.MarketingOptIn, &p.CreatedAt)

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

	// Disable SMS notifications by default — users must explicitly opt in.
	if err := UpsertNotificationPreference(ctx, db, userID, "sms", false); err != nil {
		return nil, err
	}

	return &p, nil
}
