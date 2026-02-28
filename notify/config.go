package notify

import (
	"log"
	"os"
	"strings"
)

// Config holds notification-related configuration loaded from environment
// variables at startup. Individual channel configs (SendGrid API key, Twilio
// SID, VAPID keys, etc.) are defined by their respective channel packages
// and added to this struct as they're implemented.
type Config struct {
	// Email channel — exactly one provider is used (SendGrid preferred).
	EmailProvider  string // "twilio-sendgrid" or "smtp"; empty = disabled
	EmailFrom      string // RFC-5321 sender address for all outbound email
	SendGridAPIKey string // required when EmailProvider == "twilio-sendgrid"
	SMTPHost       string // required when EmailProvider == "smtp"
	SMTPPort       string // defaults to "587"
	SMTPUsername   string
	SMTPPassword   string

	// SMS (Twilio) — issue #277
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string

	// Web Push (VAPID) — issue #276
	// VAPIDSubject is a mailto: URL identifying the server operator, required
	// by the Web Push spec. Example: "mailto:admin@mycompany.com"
	VAPIDSubject string
}

// LoadConfig reads notification-related environment variables and returns a
// Config. It does not fail — missing env vars simply mean the corresponding
// channel won't be configured.
func LoadConfig() Config {
	return Config{
		EmailProvider:    os.Getenv("NOTIFICATION_EMAIL_PROVIDER"),
		EmailFrom:        os.Getenv("NOTIFICATION_EMAIL_FROM"),
		SendGridAPIKey:   os.Getenv("SENDGRID_API_KEY"),
		SMTPHost:         os.Getenv("SMTP_HOST"),
		SMTPPort:         EnvOrDefault("SMTP_PORT", "587"),
		SMTPUsername:     os.Getenv("SMTP_USERNAME"),
		SMTPPassword:     os.Getenv("SMTP_PASSWORD"),
		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFromNumber: os.Getenv("TWILIO_FROM_NUMBER"),
		VAPIDSubject:     EnvOrDefault("VAPID_SUBJECT", ""),
	}
}

// BuildSenders inspects the config and returns the set of Sender
// implementations whose required env vars are present. Each channel issue
// (#275, #276, #277) will add its sender construction here.
//
// Returns nil when no channels are configured.
func (c Config) BuildSenders() []Sender {
	var senders []Sender

	// #275 — Email (SendGrid + SMTP)
	if s := c.buildEmailSender(); s != nil {
		senders = append(senders, s)
	}

	// Channel registrations for future issues:
	//
	// #276 — Web Push (VAPID):
	//   if c.VAPIDPrivateKey != "" { senders = append(senders, webpush.New(...)) }

	// #277 — SMS (Twilio)
	if c.TwilioAccountSID != "" && c.TwilioAuthToken != "" && c.TwilioFromNumber != "" {
		senders = append(senders, NewSMSSender(SMSConfig{
			AccountSID: c.TwilioAccountSID,
			AuthToken:  c.TwilioAuthToken,
			FromNumber: c.TwilioFromNumber,
		}))
	} else if c.TwilioAccountSID != "" || c.TwilioAuthToken != "" || c.TwilioFromNumber != "" {
		log.Println("notify: SMS (Twilio) partially configured — set all three: TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER")
	}

	return senders
}

// buildEmailSender returns an email Sender based on the configured provider,
// or nil when email is not configured.
func (c Config) buildEmailSender() Sender {
	switch c.EmailProvider {
	case "twilio-sendgrid":
		if c.SendGridAPIKey == "" {
			log.Println("Notifications: NOTIFICATION_EMAIL_PROVIDER=twilio-sendgrid but SENDGRID_API_KEY is not set — email disabled")
			return nil
		}
		if c.EmailFrom == "" {
			log.Println("Notifications: NOTIFICATION_EMAIL_PROVIDER=twilio-sendgrid but NOTIFICATION_EMAIL_FROM is not set — email disabled")
			return nil
		}
		if containsCRLF(c.EmailFrom) {
			log.Println("Notifications: NOTIFICATION_EMAIL_FROM contains invalid characters (CR/LF) — email disabled")
			return nil
		}
		log.Printf("Notifications: email channel enabled (SendGrid, from=%s)", c.EmailFrom)
		return NewSendGridSender(c.SendGridAPIKey, c.EmailFrom)

	case "smtp":
		if c.SMTPHost == "" {
			log.Println("Notifications: NOTIFICATION_EMAIL_PROVIDER=smtp but SMTP_HOST is not set — email disabled")
			return nil
		}
		if c.EmailFrom == "" {
			log.Println("Notifications: NOTIFICATION_EMAIL_PROVIDER=smtp but NOTIFICATION_EMAIL_FROM is not set — email disabled")
			return nil
		}
		if containsCRLF(c.EmailFrom) {
			log.Println("Notifications: NOTIFICATION_EMAIL_FROM contains invalid characters (CR/LF) — email disabled")
			return nil
		}
		log.Printf("Notifications: email channel enabled (SMTP %s:%s, from=%s)", c.SMTPHost, c.SMTPPort, c.EmailFrom)
		return NewSMTPSender(c.SMTPHost, c.SMTPPort, c.SMTPUsername, c.SMTPPassword, c.EmailFrom)

	case "":
		// Not configured — no warning here; LogChannelSummary handles the aggregate warning.
		return nil

	default:
		log.Printf("Notifications: unknown NOTIFICATION_EMAIL_PROVIDER=%q (expected \"twilio-sendgrid\" or \"smtp\") — email disabled", c.EmailProvider)
		return nil
	}
}

// LogChannelSummary logs which notification channels are configured at
// startup. Warns when zero channels are present.
func LogChannelSummary(senders []Sender) {
	if len(senders) == 0 {
		log.Println("Notifications: no channels configured (set channel env vars to enable — see .env.example)")
		return
	}
	names := make([]string, len(senders))
	for i, s := range senders {
		names[i] = s.Name()
	}
	log.Printf("Notifications: %d channel(s) configured: %v", len(senders), names)
}

// EnvOrDefault returns the value of the named environment variable, or
// fallback if unset/empty. Useful for channel packages that need defaults.
func EnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// containsCRLF returns true if s contains \r or \n. Used to reject env var
// values that could cause SMTP header injection.
func containsCRLF(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}
