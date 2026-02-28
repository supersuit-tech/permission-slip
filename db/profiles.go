package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Profile represents a row from the profiles table.
type Profile struct {
	ID        string
	Username  string
	Email     *string // nullable — user opts in by setting an address
	Phone     *string // nullable — E.164 format (e.g. "+15551234567")
	CreatedAt time.Time
}

// GetProfileByUserID returns the profile for the given user ID,
// or nil if no profile exists.
func GetProfileByUserID(ctx context.Context, db DBTX, userID string) (*Profile, error) {
	var p Profile
	err := db.QueryRow(ctx,
		"SELECT id, username, email, phone, created_at FROM profiles WHERE id = $1",
		userID,
	).Scan(&p.ID, &p.Username, &p.Email, &p.Phone, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdateProfileContactFields updates the email and phone columns for the
// given user. Pass nil to clear a field.
func UpdateProfileContactFields(ctx context.Context, db DBTX, userID string, email, phone *string) error {
	_, err := db.Exec(ctx,
		"UPDATE profiles SET email = $2, phone = $3 WHERE id = $1",
		userID, email, phone,
	)
	return err
}
