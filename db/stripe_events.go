package db

import "context"

// RecordStripeEvent inserts a processed Stripe webhook event ID.
// Returns false if the event was already recorded (duplicate).
// This provides idempotency for webhook handlers — if a handler fails
// before recording the event, Stripe's retry will reprocess it.
func RecordStripeEvent(ctx context.Context, db DBTX, eventID, eventType string) (bool, error) {
	tag, err := db.Exec(ctx,
		`INSERT INTO stripe_webhook_events (event_id, event_type)
		 VALUES ($1, $2)
		 ON CONFLICT (event_id) DO NOTHING`,
		eventID, eventType)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// IsStripeEventProcessed checks if a Stripe webhook event has already been processed.
func IsStripeEventProcessed(ctx context.Context, db DBTX, eventID string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM stripe_webhook_events WHERE event_id = $1)`,
		eventID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// PurgeOldStripeEvents deletes events older than 72 hours.
// Stripe retries webhooks for up to 72 hours, so events older than that
// are safe to purge.
func PurgeOldStripeEvents(ctx context.Context, db DBTX) (int64, error) {
	tag, err := db.Exec(ctx,
		`DELETE FROM stripe_webhook_events
		 WHERE processed_at < now() - interval '72 hours'`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

