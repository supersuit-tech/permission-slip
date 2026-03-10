package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// FindProfileByAuthEmail looks up a profile whose auth.users entry has the
// given email address. This is used as a fallback when a returning user's
// Supabase identity gets a new UUID for the same email — allowing us to
// find their existing profile and re-link it.
//
// Returns nil, nil if no matching profile exists or if the auth.users table
// lacks an email column (e.g. minimal local-dev stub before migration).
func FindProfileByAuthEmail(ctx context.Context, db DBTX, email string) (*Profile, error) {
	if email == "" {
		return nil, nil
	}
	var p Profile
	err := db.QueryRow(ctx,
		`SELECT p.id, p.username, p.email, p.phone, p.marketing_opt_in, p.created_at
		 FROM profiles p
		 JOIN auth.users au ON au.id = p.id
		 WHERE au.email = $1
		 LIMIT 1`,
		email,
	).Scan(&p.ID, &p.Username, &p.Email, &p.Phone, &p.MarketingOptIn, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		// 42703 = undefined_column: the auth.users table lacks an email
		// column in environments that haven't run the migration yet.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42703" {
			return nil, nil
		}
		return nil, err
	}
	p.CreatedAt = p.CreatedAt.UTC().Truncate(time.Millisecond)
	return &p, nil
}

// RelinkProfile updates a profile's primary key from oldID to newID.
// All child tables with ON UPDATE CASCADE follow automatically.
// The new user must already exist in auth.users (Supabase creates it on login).
//
// In local dev, auth.users may not have the new ID yet, so we insert it
// first (id only, no email — avoids unique constraint conflicts on email).
// This bare (id) INSERT works because the local-dev auth.users stub
// (created by testhelper/migrations) has only nullable columns besides id.
// In production, auth.users has NOT NULL columns (aud, role, etc.), but
// the INSERT is always a no-op there because Supabase already created the
// row during OTP login.
// If step 1 succeeds but step 2 fails, the orphaned auth.users row is
// harmless and the next request will retry the full operation.
func RelinkProfile(ctx context.Context, db DBTX, oldID, newID string) error {
	// Ensure the new auth.users entry exists (Supabase manages this in prod;
	// in local dev we create a minimal row ourselves). We omit the email
	// to avoid colliding with the old user's unique email constraint.
	//
	// In production the app_backend role only has SELECT on auth.users
	// (INSERT is intentionally omitted because Supabase creates the row
	// during OTP login). The INSERT is therefore always a no-op in prod,
	// but PostgreSQL still checks INSERT privilege before evaluating
	// ON CONFLICT DO NOTHING — resulting in a 42501 (insufficient_privilege)
	// error. We treat this specific error as success: if we lack INSERT
	// permission, the row must already exist (managed by Supabase).
	_, err := db.Exec(ctx,
		`INSERT INTO auth.users (id) VALUES ($1) ON CONFLICT DO NOTHING`,
		newID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42501" {
			// insufficient_privilege — expected in production where
			// app_backend lacks INSERT on auth.users. Safe to ignore
			// because Supabase already created the row during login.
		} else {
			return err
		}
	}

	// Update the profile PK — ON UPDATE CASCADE propagates to all child tables.
	ct, err := db.Exec(ctx,
		`UPDATE profiles SET id = $1 WHERE id = $2`,
		newID, oldID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New("profile not found during re-link")
	}
	return nil
}

// FindProfileByUsername returns the profile with the given username,
// or nil if no such profile exists.
func FindProfileByUsername(ctx context.Context, db DBTX, username string) (*Profile, error) {
	var p Profile
	err := db.QueryRow(ctx,
		`SELECT id, username, email, phone, marketing_opt_in, created_at
		 FROM profiles WHERE username = $1`,
		username,
	).Scan(&p.ID, &p.Username, &p.Email, &p.Phone, &p.MarketingOptIn, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.CreatedAt = p.CreatedAt.UTC().Truncate(time.Millisecond)
	return &p, nil
}
