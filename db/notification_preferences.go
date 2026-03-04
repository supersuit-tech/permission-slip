package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// NotificationPreference represents a row from the notification_preferences table.
type NotificationPreference struct {
	UserID  string
	Channel string // "email", "web-push", "sms", or "mobile-push"
	Enabled bool
}

// GetNotificationPreferences returns all notification preferences for the user.
// Channels without a row are implicitly enabled (callers should treat missing
// channels as enabled).
func GetNotificationPreferences(ctx context.Context, db DBTX, userID string) ([]NotificationPreference, error) {
	rows, err := db.Query(ctx,
		"SELECT user_id, channel, enabled FROM notification_preferences WHERE user_id = $1",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []NotificationPreference
	for rows.Next() {
		var p NotificationPreference
		if err := rows.Scan(&p.UserID, &p.Channel, &p.Enabled); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, rows.Err()
}

// IsNotificationChannelEnabled checks whether a specific channel is enabled
// for the user. A missing preference row defaults to true (enabled).
func IsNotificationChannelEnabled(ctx context.Context, db DBTX, userID, channel string) (bool, error) {
	var enabled bool
	err := db.QueryRow(ctx,
		"SELECT enabled FROM notification_preferences WHERE user_id = $1 AND channel = $2",
		userID, channel,
	).Scan(&enabled)
	if err == pgx.ErrNoRows {
		return true, nil // missing row → default enabled
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// UpsertNotificationPreference inserts or updates a single channel preference.
func UpsertNotificationPreference(ctx context.Context, db DBTX, userID, channel string, enabled bool) error {
	_, err := db.Exec(ctx,
		`INSERT INTO notification_preferences (user_id, channel, enabled)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, channel)
		 DO UPDATE SET enabled = EXCLUDED.enabled`,
		userID, channel, enabled,
	)
	return err
}
