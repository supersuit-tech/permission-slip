package db

import (
	"context"
	"time"
)

// PushSubscription represents a row from the push_subscriptions table.
type PushSubscription struct {
	ID        int64
	UserID    string
	Endpoint  string
	P256dh    string // base64url-encoded P-256 public key
	Auth      string // base64url-encoded auth secret
	CreatedAt time.Time
}

// UpsertPushSubscription inserts a push subscription or updates it if the
// (user_id, endpoint) pair already exists (browser re-subscribed).
func UpsertPushSubscription(ctx context.Context, db DBTX, userID, endpoint, p256dh, auth string) (*PushSubscription, error) {
	var sub PushSubscription
	err := db.QueryRow(ctx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, endpoint)
		 DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth
		 RETURNING id, user_id, endpoint, p256dh, auth, created_at`,
		userID, endpoint, p256dh, auth,
	).Scan(&sub.ID, &sub.UserID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.CreatedAt)
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

// ListPushSubscriptionsByUserID returns all push subscriptions for a user.
func ListPushSubscriptionsByUserID(ctx context.Context, db DBTX, userID string) ([]PushSubscription, error) {
	rows, err := db.Query(ctx,
		"SELECT id, user_id, endpoint, p256dh, auth, created_at FROM push_subscriptions WHERE user_id = $1 ORDER BY created_at",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []PushSubscription
	for rows.Next() {
		var s PushSubscription
		if err := rows.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}
