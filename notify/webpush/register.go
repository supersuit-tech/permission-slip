package webpush

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/notify"
)

func init() {
	notify.RegisterSenderFactory(func(ctx context.Context, bc notify.BuildContext) ([]notify.Sender, error) {
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
			log.Fatalf("Failed to initialize VAPID keys: %v", err)
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
				log.Fatalf("Web Push: VAPID_SUBJECT is required in production (e.g. mailto:admin@mycompany.com or https://example.com/contact)")
			}
		}

		return []notify.Sender{New(vapidKeys, subject, bc.DB)}, nil
	})
}
