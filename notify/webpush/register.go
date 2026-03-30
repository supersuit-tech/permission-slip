package webpush

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/notify"
)

func init() {
	notify.RegisterSenderFactory("web-push", func(ctx context.Context, bc notify.BuildContext) ([]notify.Sender, error) {
		if bc.DB == nil {
			return nil, nil
		}

		vapidCtx, vapidCancel := context.WithTimeout(ctx, 10*time.Second)
		vapidKeys, err := InitVAPIDKeys(vapidCtx, bc.DB, bc.DevMode)
		vapidCancel()
		if err != nil {
			if bc.DevMode {
				log.Printf("Warning: failed to initialize VAPID keys: %v", err)
				return nil, nil
			}
			// Return an error so BuildSenders logs it and skips web push.
			// In production the startup validator (validateConfig) already
			// ensures VAPID env vars are present, so this path indicates an
			// unexpected database failure rather than misconfiguration.
			return nil, fmt.Errorf("initialize VAPID keys: %w", err)
		}
		if vapidKeys == nil {
			return nil, nil
		}

		if bc.OnVAPIDPublicKey != nil {
			bc.OnVAPIDPublicKey(vapidKeys.PublicKey)
		}

		subject := strings.TrimSpace(bc.Config.VAPIDSubject)
		if subject == "" {
			if bc.DevMode {
				subject = "mailto:admin@example.com"
				log.Println("Web Push: VAPID_SUBJECT not set, using default mailto:admin@example.com (development mode only)")
			} else {
				// Return an error rather than calling log.Fatalf so that
				// BuildSenders' error-handling contract is respected. In
				// production validateConfig already catches a missing
				// VAPID_SUBJECT before the server starts.
				return nil, fmt.Errorf("VAPID_SUBJECT is required in production (e.g. mailto:admin@mycompany.com or https://example.com/contact)")
			}
		}

		return []notify.Sender{New(vapidKeys, subject, bc.DB)}, nil
	})
}
