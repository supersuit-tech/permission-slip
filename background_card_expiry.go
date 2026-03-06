package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/notify"
)

// CardExpiryCheckDeps holds the dependencies for the background card
// expiration check job.
type CardExpiryCheckDeps struct {
	DB       db.DBTX
	Notifier *notify.Dispatcher
	BaseURL  string
}

// startCardExpiryCheck runs a periodic background job that detects payment
// methods expiring within 30 days (or already expired) and sends a one-time
// notification via the existing notification system. It returns a channel
// that is closed when the goroutine exits for clean shutdown coordination.
func startCardExpiryCheck(ctx context.Context, deps CardExpiryCheckDeps, logger *slog.Logger) <-chan struct{} {
	interval := cardExpiryCheckInterval(logger)
	logger.Info("card expiry check: scheduled", "interval", interval.String())

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Run once immediately on startup to catch up after downtime.
		runCardExpiryCheck(ctx, deps, logger)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("card expiry check: stopped")
				return
			case <-ticker.C:
				runCardExpiryCheck(ctx, deps, logger)
			}
		}
	}()
	return done
}

// runCardExpiryCheck executes a single pass: finds expiring cards, sends
// notifications, and marks them as alerted so they aren't re-notified.
func runCardExpiryCheck(ctx context.Context, deps CardExpiryCheckDeps, logger *slog.Logger) {
	checkCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	expiring, err := db.ListExpiringPaymentMethods(checkCtx, deps.DB, 30)
	if err != nil {
		if ctx.Err() != nil {
			logger.Info("card expiry check: cancelled", "error", err)
			return
		}
		logger.Error("card expiry check: failed to list expiring cards", "error", err)
		sentry.CaptureException(err)
		return
	}

	if len(expiring) == 0 {
		logger.Debug("card expiry check: no expiring cards found")
		return
	}

	logger.Info("card expiry check: found expiring cards", "count", len(expiring))

	now := time.Now()
	settingsURL := ""
	if deps.BaseURL != "" {
		settingsURL = deps.BaseURL + "/settings?tab=billing"
	}

	for _, epm := range expiring {
		pm := epm.PaymentMethod
		profile := epm.Profile

		// Determine if the card is already expired
		expired := isCardExpired(pm.ExpMonth, pm.ExpYear, now)

		// Build context JSON with card details for templates
		cardInfo := notify.CardExpiringInfo{
			Brand:    pm.Brand,
			Last4:    pm.Last4,
			Label:    pm.Label,
			ExpMonth: pm.ExpMonth,
			ExpYear:  pm.ExpYear,
			Expired:  expired,
		}
		contextJSON, _ := json.Marshal(cardInfo)

		approval := notify.Approval{
			ApprovalID:  pm.ID, // use payment method ID as a stable identifier
			ApprovalURL: settingsURL,
			Type:        notify.NotificationTypeCardExpiring,
			Context:     contextJSON,
			CreatedAt:   now,
			ExpiresAt:   now.Add(30 * 24 * time.Hour), // no real expiry; set far out
		}

		recipient := notify.Recipient{
			UserID:   profile.ID,
			Username: profile.Username,
			Email:    profile.Email,
			Phone:    profile.Phone,
		}

		// Use DispatchSync so we can mark the alert as sent only after
		// delivery completes (best-effort). This is a background job,
		// not an HTTP handler, so blocking is fine.
		deps.Notifier.DispatchSync(checkCtx, approval, recipient)

		// Mark the card as alerted to prevent duplicate notifications.
		if err := db.MarkExpirationAlertSent(checkCtx, deps.DB, pm.ID); err != nil {
			logger.Error("card expiry check: failed to mark alert sent",
				"payment_method_id", pm.ID, "error", err)
			sentry.CaptureException(err)
		} else {
			logger.Info("card expiry check: alert sent",
				"payment_method_id", pm.ID,
				"brand", pm.Brand,
				"last4", pm.Last4,
				"expired", expired,
				"user_id", profile.ID)
		}
	}
}

// isCardExpired returns true if the card's expiry month/year is in the past.
// Cards expire at the end of their expiry month.
func isCardExpired(expMonth, expYear int, now time.Time) bool {
	currentMonth := int(now.Month())
	currentYear := now.Year()
	return expYear < currentYear || (expYear == currentYear && expMonth < currentMonth)
}

// cardExpiryCheckInterval returns the check interval from the
// CARD_EXPIRY_CHECK_INTERVAL env var, defaulting to 24 hours (daily).
// Values below 1 hour are rejected to prevent resource waste.
func cardExpiryCheckInterval(logger *slog.Logger) time.Duration {
	const minInterval = time.Hour

	if v := os.Getenv("CARD_EXPIRY_CHECK_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d >= minInterval {
			return d
		}
		logger.Warn("invalid CARD_EXPIRY_CHECK_INTERVAL, using default 24h",
			"value", v, "error", err, "min", minInterval.String())
	}
	return 24 * time.Hour
}
