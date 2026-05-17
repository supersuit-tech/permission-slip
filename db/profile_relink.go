package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// FindProfileByAuthEmail is a no-op stub left in place during the cut-over
// from Supabase to local auth. The original implementation handled the
// Supabase-specific case where a returning user got a new auth UUID for the
// same email; with locally-managed users that ID is stable so re-linking is
// unnecessary. Removed entirely in Phase 2 along with its single caller in
// api/session.go.
func FindProfileByAuthEmail(_ context.Context, _ DBTX, _ string) (*Profile, error) {
	return nil, nil
}

// RelinkProfile is a no-op stub for the same reason as FindProfileByAuthEmail.
// Removed in Phase 2.
func RelinkProfile(_ context.Context, _ DBTX, _, _ string) error {
	return nil
}

// FindProfileByUsername returns the profile with the given username, or nil
// if no such profile exists.
func FindProfileByUsername(ctx context.Context, db DBTX, username string) (*Profile, error) {
	var p Profile
	err := db.QueryRow(ctx,
		`SELECT id, username, email, phone, marketing_opt_in, created_at
		 FROM profiles WHERE username = $1`,
		username,
	).Scan(&p.ID, &p.Username, &p.Email, &p.Phone, &p.MarketingOptIn, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt = p.CreatedAt.UTC().Truncate(time.Millisecond)
	return &p, nil
}
