package notify

import (
	"context"
	"log"
	"os"
	"strings"
)

// Config holds notification-related configuration loaded from environment
// variables at startup. Individual channel configs (SendGrid API key, AWS
// credentials, VAPID keys, etc.) are defined by their respective channel
// packages and added to this struct as they're implemented.
type Config struct {
	// Email channel — exactly one provider is used (SendGrid preferred).
	EmailProvider  string // "twilio-sendgrid" or "smtp"; empty = disabled
	EmailFrom      string // RFC-5321 sender address for all outbound email
	SendGridAPIKey string // required when EmailProvider == "twilio-sendgrid"
	SMTPHost       string // required when EmailProvider == "smtp"
	SMTPPort       string // defaults to "587"
	SMTPUsername   string
	SMTPPassword   string

	// SMS (Amazon SNS) — issue #690
	AWSRegion            string
	AWSAccessKeyID       string
	AWSSecretAccessKey   string
	SNSSenderID          string // optional alphanumeric sender ID
	SNSOriginationNumber string // optional origination number (E.164)

	// Web Push (VAPID) — issue #276
	// VAPIDSubject is a mailto: URL identifying the server operator, required
	// by the Web Push spec. Example: "mailto:admin@mycompany.com"
	VAPIDSubject string

	// Mobile Push (Expo) — issue #9
	// ExpoAccessToken is an optional Expo access token for authenticated
	// requests to the Expo Push Service. When empty, unauthenticated
	// requests are used (lower rate limits).
	ExpoAccessToken string
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
		AWSRegion:            os.Getenv("AWS_REGION"),
		AWSAccessKeyID:       os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey:   os.Getenv("AWS_SECRET_ACCESS_KEY"),
		SNSSenderID:          os.Getenv("SNS_SMS_SENDER_ID"),
		SNSOriginationNumber: os.Getenv("SNS_SMS_ORIGINATION_NUMBER"),
		VAPIDSubject:     EnvOrDefault("VAPID_SUBJECT", ""),
		ExpoAccessToken:  os.Getenv("EXPO_ACCESS_TOKEN"),
	}
}

// BuildSenders calls the globally registered sender factories and returns the
// set of Sender implementations whose required config is present. Each channel
// registers itself via init() in its own package.
//
// This method is provided for backward-compatibility and testing convenience.
// Production code should prefer calling notify.BuildSenders(ctx, BuildContext{...})
// with the full build context, including the database handle (required for web
// push, mobile push, and SMS plan gating).
//
// Returns nil when no channels are configured.
func (c Config) BuildSenders() []Sender {
	return BuildSenders(context.Background(), BuildContext{Config: c})
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
