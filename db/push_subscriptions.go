package db

import (
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

// UpsertPushSubscription inserts a web-push subscription or updates it if the
// (user_id, endpoint) pair already exists (browser re-subscribed).
func UpsertPushSubscription(ctx context.Context, db DBTX, userID, endpoint, p256dh, auth string) (*PushSubscription, error) {
	var sub PushSubscription
	err := db.QueryRow(ctx,
		`INSERT INTO push_subscriptions (user_id, channel, endpoint, p256dh, auth)
		 VALUES ($1, 'web-push', $2, $3, $4)
		 ON CONFLICT (user_id, endpoint) WHERE endpoint IS NOT NULL
		 DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth
		 RETURNING id, user_id, channel, endpoint, p256dh, auth, expo_token, created_at`,
		userID, endpoint, p256dh, auth,
	).Scan(&sub.ID, &sub.UserID, &sub.Channel, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.ExpoToken, &sub.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// UpsertExpoPushToken inserts an Expo push token subscription or updates it
// if the (user_id, expo_token) pair already exists (device re-registered).
func UpsertExpoPushToken(ctx context.Context, db DBTX, userID, expoToken string) (*PushSubscription, error) {
	var sub PushSubscription
	err := db.QueryRow(ctx,
		`INSERT INTO push_subscriptions (user_id, channel, expo_token)
		 VALUES ($1, 'mobile-push', $2)
		 ON CONFLICT (user_id, expo_token) WHERE expo_token IS NOT NULL
		 DO UPDATE SET expo_token = EXCLUDED.expo_token
		 RETURNING id, user_id, channel, endpoint, p256dh, auth, expo_token, created_at`,
		userID, expoToken,
	).Scan(&sub.ID, &sub.UserID, &sub.Channel, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.ExpoToken, &sub.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sub, nil
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
	return tag.RowsAffected() > 0, nil
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

// DeleteExpoPushToken removes a push subscription by Expo token.
// Used when the Expo push service reports a token as invalid.
func DeleteExpoPushToken(ctx context.Context, db DBTX, expoToken string) error {
	_, err := db.Exec(ctx,
		"DELETE FROM push_subscriptions WHERE expo_token = $1",
		expoToken,
	)
	return err
}

// ListPushSubscriptionsByUserID returns all push subscriptions for a user.
func ListPushSubscriptionsByUserID(ctx context.Context, db DBTX, userID string) ([]PushSubscription, error) {
	rows, err := db.Query(ctx,
		"SELECT id, user_id, channel, endpoint, p256dh, auth, expo_token, created_at FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Channel, &s.Endpoint, &s.P256dh, &s.Auth, &s.ExpoToken, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// ListWebPushSubscriptionsByUserID returns only web-push subscriptions for a user.
func ListWebPushSubscriptionsByUserID(ctx context.Context, db DBTX, userID string) ([]PushSubscription, error) {
	rows, err := db.Query(ctx,
		"SELECT id, user_id, channel, endpoint, p256dh, auth, expo_token, created_at FROM push_subscriptions WHERE user_id = $1 AND channel = 'web-push' ORDER BY created_at",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Channel, &s.Endpoint, &s.P256dh, &s.Auth, &s.ExpoToken, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

// ListExpoPushTokensByUserID returns only mobile-push (Expo) subscriptions for a user.
func ListExpoPushTokensByUserID(ctx context.Context, db DBTX, userID string) ([]PushSubscription, error) {
	rows, err := db.Query(ctx,
		"SELECT id, user_id, channel, endpoint, p256dh, auth, expo_token, created_at FROM push_subscriptions WHERE user_id = $1 AND channel = 'mobile-push' ORDER BY created_at",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Channel, &s.Endpoint, &s.P256dh, &s.Auth, &s.ExpoToken, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
