package notify

import (
	"context"
	"log"
)

func init() {
	RegisterSenderFactory(func(_ context.Context, bc BuildContext) ([]Sender, error) {
		cfg := bc.Config
		if cfg.TwilioAccountSID == "" || cfg.TwilioAuthToken == "" || cfg.TwilioFromNumber == "" {
			if cfg.TwilioAccountSID != "" || cfg.TwilioAuthToken != "" || cfg.TwilioFromNumber != "" {
				log.Println("notify: SMS (Twilio) partially configured — set all three: TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER")
			}
			return nil, nil
		}
		var s Sender = NewSMSSender(SMSConfig{
			AccountSID: cfg.TwilioAccountSID,
			AuthToken:  cfg.TwilioAuthToken,
			FromNumber: cfg.TwilioFromNumber,
		})
		// Gate SMS behind paid tier when the database is available.
		// Free-tier users are blocked; paid-tier usage is tracked in usage_periods.
		//
		// Security note: when bc.DB is nil (no database configured), plan gating
		// is skipped and SMS is sent to all recipients without billing checks.
		// This matches the behaviour of the previous inline main.go setup and only
		// occurs in configurations that explicitly disable the database.
		if bc.DB != nil {
			s = NewPlanGatedSender(s, &DBSMSGate{DB: bc.DB})
		} else {
			log.Println("notify: SMS plan gating disabled — no database configured; all recipients will receive SMS regardless of subscription tier")
		}
		return []Sender{s}, nil
	})
}
