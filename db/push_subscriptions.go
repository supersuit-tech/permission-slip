package db

import (
	"database/sql"
	"context"
	"time"

)

// Push subscription channel values.
const (
	PushChannelWebPush    = "web-push"
	PushChannelMobilePush = "mobile-push"
)

// PushSubscription represents a row from the push_subscriptions table.
// For web-push subscriptions, Endpoint/P256dh/Auth are set.
// For mobile-push (Expo) subscriptions, ExpoToken is set.
type PushSubscription struct {
	ID        int64
	UserID    string
	Channel   string
	Endpoint  *string // Web Push endpoint URL (nil for mobile-push)
	P256dh    *string // base64url-encoded P-256 public key (nil for mobile-push)
	Auth      *string // base64url-encoded auth secret (nil for mobile-push)
	ExpoToken *string // Expo push token (nil for web-push)
	CreatedAt time.Time
}

func scanPushSubscription(row rowScanner) (*PushSubscription, error) {
	var s PushSubscription
	var createdAt sql.NullString
	err := row.Scan(&s.ID, &s.UserID, &s.Channel, &s.Endpoint, &s.P256dh, &s.Auth, &s.ExpoToken, &createdAt)
	if err != nil {
		return nil, err
	}
	var err2 error
	s.CreatedAt, err2 = sqliteTimeRequired(createdAt)
	if err2 != nil {
		return nil, err2
	}
	return &s, nil
}

// UpsertPushSubscription inserts a web-push subscription or updates it if the
// (user_id, endpoint) pair already exists (browser re-subscribed).
func UpsertPushSubscription(ctx context.Context, db DBTX, userID, endpoint, p256dh, auth string) (*PushSubscription, error) {
	return scanPushSubscription(db.QueryRow(ctx,
		`INSERT INTO push_subscriptions (user_id, channel, endpoint, p256dh, auth)
		 VALUES ($1, 'web-push', $2, $3, $4)
		 ON CONFLICT (user_id, endpoint) WHERE endpoint IS NOT NULL
		 DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth
		 RETURNING id, user_id, channel, endpoint, p256dh, auth, expo_token, created_at`,
		userID, endpoint, p256dh, auth,
	))
}

// DeletePushSubscription removes a push subscription by ID, scoped to the user.
// Returns true if a row was deleted.
func DeletePushSubscription(ctx context.Context, db DBTX, userID string, subID int64) (bool, error) {
	tag, err := db.Exec(ctx,
		"DELETE FROM push_subscriptions WHERE id = $1 AND user_id = $2",
		subID, userID,
	)
	if err != nil {
		return false, err
	}
	return RowsAffected(tag) > 0, nil
}

// DeletePushSubscriptionByEndpoint removes a push subscription by endpoint URL.
// Used when the push service returns 410 Gone (subscription expired).
func DeletePushSubscriptionByEndpoint(ctx context.Context, db DBTX, endpoint string) error {
	_, err := db.Exec(ctx,
		"DELETE FROM push_subscriptions WHERE endpoint = $1",
		endpoint,
	)
	return err
}

// pushSubscriptionColumns is the SELECT column list shared across queries.
const pushSubscriptionColumns = "id, user_id, channel, endpoint, p256dh, auth, expo_token, created_at"

// scanPushSubscriptions reads rows into a slice of PushSubscription.
func scanPushSubscriptions(rows *sql.Rows, err error) ([]PushSubscription, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		s, err := scanPushSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, *s)
	}
	return subs, rows.Err()
}

// ListWebPushSubscriptionsByUserID returns only web-push subscriptions for a user.
func ListWebPushSubscriptionsByUserID(ctx context.Context, db DBTX, userID string) ([]PushSubscription, error) {
	return scanPushSubscriptions(db.Query(ctx,
		"SELECT "+pushSubscriptionColumns+" FROM push_subscriptions WHERE user_id = $1 AND channel = 'web-push' ORDER BY created_at",
		userID,
	))
}

