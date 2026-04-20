package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// NotificationTypeStandingExecution is stored in notification_type_preferences.notification_type
// for standing-approval (auto-approval) execution notifications.
const NotificationTypeStandingExecution = "standing_execution"

// NotificationTypePreference represents a row from notification_type_preferences.
type NotificationTypePreference struct {
	UserID           string
	NotificationType string
	Enabled          bool
}

// GetNotificationTypePreferences returns all notification type preference rows for the user.
func GetNotificationTypePreferences(ctx context.Context, db DBTX, userID string) ([]NotificationTypePreference, error) {
	rows, err := db.Query(ctx,
		"SELECT user_id, notification_type, enabled FROM notification_type_preferences WHERE user_id = $1",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []NotificationTypePreference
	for rows.Next() {
		var p NotificationTypePreference
		if err := rows.Scan(&p.UserID, &p.NotificationType, &p.Enabled); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, rows.Err()
}

// IsNotificationTypeEnabled returns whether notifications for the given logical type are enabled.
// A missing preference row defaults to true (enabled).
func IsNotificationTypeEnabled(ctx context.Context, db DBTX, userID, notificationType string) (bool, error) {
	var enabled bool
	err := db.QueryRow(ctx,
		"SELECT enabled FROM notification_type_preferences WHERE user_id = $1 AND notification_type = $2",
		userID, notificationType,
	).Scan(&enabled)
	if err == pgx.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// UpsertNotificationTypePreference inserts or updates a single notification type preference.
func UpsertNotificationTypePreference(ctx context.Context, db DBTX, userID, notificationType string, enabled bool) error {
	_, err := db.Exec(ctx,
		`INSERT INTO notification_type_preferences (user_id, notification_type, enabled)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, notification_type)
		 DO UPDATE SET enabled = EXCLUDED.enabled`,
		userID, notificationType, enabled,
	)
	return err
}
