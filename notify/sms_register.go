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
		if bc.DB != nil {
			s = NewPlanGatedSender(s, &DBSMSGate{DB: bc.DB})
		}
		return []Sender{s}, nil
	})
}
