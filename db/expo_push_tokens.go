package db

import (
	"context"
	"time"
)

// ExpoPushToken represents a row from the expo_push_tokens table.
type ExpoPushToken struct {
	ID        int64
	UserID    string
	Token     string // Expo push token, e.g. "ExponentPushToken[...]"
	CreatedAt time.Time
}

// UpsertExpoPushToken inserts an Expo push token or updates it if the
// (user_id, token) pair already exists (device re-registered).
func UpsertExpoPushToken(ctx context.Context, db DBTX, userID, token string) (*ExpoPushToken, error) {
	var t ExpoPushToken
	err := db.QueryRow(ctx,
		`INSERT INTO expo_push_tokens (user_id, token)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, token)
		 DO UPDATE SET token = EXCLUDED.token
		 RETURNING id, user_id, token, created_at`,
		userID, token,
	).Scan(&t.ID, &t.UserID, &t.Token, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteExpoPushToken removes an Expo push token by ID, scoped to the user.
// Returns true if a row was deleted.
func DeleteExpoPushToken(ctx context.Context, db DBTX, userID string, tokenID int64) (bool, error) {
	tag, err := db.Exec(ctx,
		"DELETE FROM expo_push_tokens WHERE id = $1 AND user_id = $2",
		tokenID, userID,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteExpoPushTokenForUser removes an Expo push token by its token string,
// scoped to a specific user. Used by the mobile app on logout/unregister.
// Returns true if a row was deleted.
func DeleteExpoPushTokenForUser(ctx context.Context, db DBTX, userID, token string) (bool, error) {
	tag, err := db.Exec(ctx,
		"DELETE FROM expo_push_tokens WHERE user_id = $1 AND token = $2",
		userID, token,
	)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteExpoPushTokenByToken removes an Expo push token by its token string.
// Used when the Expo push service reports the token as invalid.
func DeleteExpoPushTokenByToken(ctx context.Context, db DBTX, token string) error {
	_, err := db.Exec(ctx,
		"DELETE FROM expo_push_tokens WHERE token = $1",
		token,
	)
	return err
}

// ListExpoPushTokensByUserID returns all Expo push tokens for a user.
func ListExpoPushTokensByUserID(ctx context.Context, db DBTX, userID string) ([]ExpoPushToken, error) {
	rows, err := db.Query(ctx,
		"SELECT id, user_id, token, created_at FROM expo_push_tokens WHERE user_id = $1 ORDER BY created_at",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []ExpoPushToken
	for rows.Next() {
		var t ExpoPushToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Token, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
