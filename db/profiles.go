package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Profile represents a row from the profiles table.
type Profile struct {
	ID             string
	Username       string
	Email          *string // nullable — user opts in by setting an address
	Phone          *string // nullable — E.164 format (e.g. "+15551234567")
	MarketingOptIn bool    // opt-in for product update emails
	CreatedAt      time.Time
}

func scanProfile(row rowScanner) (*Profile, error) {
	var p Profile
	var createdAt sql.NullString
	err := row.Scan(&p.ID, &p.Username, &p.Email, &p.Phone, &p.MarketingOptIn, &createdAt)
	if err != nil {
		return nil, err
	}
	var err2 error
	p.CreatedAt, err2 = sqliteTimeRequired(createdAt)
	if err2 != nil {
		return nil, err2
	}
	return &p, nil
}

// GetProfileByUserID returns the profile for the given user ID,
// or nil if no profile exists.
func GetProfileByUserID(ctx context.Context, db DBTX, userID string) (*Profile, error) {
	p, err := scanProfile(db.QueryRow(ctx,
		"SELECT id, username, email, phone, marketing_opt_in, created_at FROM profiles WHERE id = $1",
		userID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

// UpdateProfileFields updates the mutable profile columns for the given user.
// Pass nil for string fields to clear them; pass nil for marketingOptIn to
// leave it unchanged.
func UpdateProfileFields(ctx context.Context, db DBTX, userID string, email, phone *string, marketingOptIn *bool) error {
	_, err := db.Exec(ctx,
		"UPDATE profiles SET email = $2, phone = $3, marketing_opt_in = COALESCE($4, marketing_opt_in) WHERE id = $1",
		userID, email, phone, marketingOptIn,
	)
	return err
}
