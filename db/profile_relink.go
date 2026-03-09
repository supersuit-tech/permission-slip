package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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
		// Gracefully handle missing email column in local-dev auth.users stub.
		// The column is added by a migration but may not exist in older envs.
		return nil, nil
	}
	return &p, nil
}

// RelinkProfile atomically updates a profile's primary key from oldID to
// newID. All child tables with ON UPDATE CASCADE follow automatically.
// The new user must already exist in auth.users (Supabase creates it on login).
func RelinkProfile(ctx context.Context, db DBTX, oldID, newID string) error {
	// Ensure the new auth.users entry exists (Supabase manages this in prod;
	// in local dev we upsert it ourselves).
	_, err := db.Exec(ctx,
		`INSERT INTO auth.users (id, email) VALUES ($1, (SELECT email FROM auth.users WHERE id = $2))
		 ON CONFLICT (id) DO NOTHING`,
		newID, oldID,
	)
	if err != nil {
		return err
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
