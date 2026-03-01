package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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

// GetProfileByUserID returns the profile for the given user ID,
// or nil if no profile exists.
func GetProfileByUserID(ctx context.Context, db DBTX, userID string) (*Profile, error) {
	var p Profile
	err := db.QueryRow(ctx,
		"SELECT id, username, email, phone, marketing_opt_in, created_at FROM profiles WHERE id = $1",
		userID,
	).Scan(&p.ID, &p.Username, &p.Email, &p.Phone, &p.MarketingOptIn, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
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
