package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// OnboardingError represents a typed error from CreateProfile.
type OnboardingError struct {
	Code    string
	Message string
}

func (e *OnboardingError) Error() string { return e.Message }

const (
	OnboardingErrUsernameTaken = "username_taken"
	OnboardingErrProfileExists = "profile_exists"
)

// CreateProfile provisions a profile for a user. The user row must already
// exist in the users table (created by the auth package's signup flow).
func CreateProfile(ctx context.Context, db DBTX, userID, username, email string, marketingOptIn bool) (*Profile, error) {
	var emailArg *string
	if email != "" {
		emailArg = &email
	}

	// profiles.id references users(id); JWT-authenticated onboarding may run
	// before any auth/signup row exists locally (tests use session-only JWT).
	stubEmail := email
	if stubEmail == "" {
		stubEmail = userID + "@onboarding.local"
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, 'onboarding-pending')
		 ON CONFLICT (id) DO NOTHING`,
		userID, stubEmail,
	); err != nil {
		return nil, fmt.Errorf("ensure user row: %w", err)
	}

	p, err := scanProfile(db.QueryRow(ctx,
		`INSERT INTO profiles (id, username, email, marketing_opt_in)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, username, email, phone, marketing_opt_in, created_at`,
		userID, username, emailArg, marketingOptIn,
	))

	if err != nil {
		if IsUniqueViolation(err) {
			// SQLite unique-violation messages include the column path,
			// e.g. "UNIQUE constraint failed: profiles.username" or
			// "UNIQUE constraint failed: profiles.id". Distinguish so the
			// caller gets a useful error code.
			if strings.Contains(err.Error(), "profiles.username") {
				return nil, &OnboardingError{
					Code:    OnboardingErrUsernameTaken,
					Message: "username is already taken",
				}
			}
			return nil, &OnboardingError{
				Code:    OnboardingErrProfileExists,
				Message: "profile already exists",
			}
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	p.CreatedAt = p.CreatedAt.UTC().Truncate(time.Millisecond)

	// Disable SMS notifications by default — users must explicitly opt in.
	if err := UpsertNotificationPreference(ctx, db, userID, "sms", false); err != nil {
		return nil, err
	}

	return p, nil
}
