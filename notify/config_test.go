package notify

import (
	"testing"
)

func TestBuildSenders_NoConfig(t *testing.T) {
	t.Parallel()
	cfg := Config{}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Errorf("expected 0 senders, got %d", len(senders))
	}
}

func TestBuildSenders_SendGrid(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider:  "twilio-sendgrid",
		SendGridAPIKey: "SG.test-key",
		EmailFrom:      "noreply@example.com",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 1 {
		t.Fatalf("expected 1 sender, got %d", len(senders))
	}
	if senders[0].Name() != "email" {
		t.Errorf("expected sender name=email, got: %s", senders[0].Name())
	}
}

func TestBuildSenders_SendGrid_MissingAPIKey(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider: "twilio-sendgrid",
		EmailFrom:     "noreply@example.com",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Errorf("expected 0 senders when API key is missing, got %d", len(senders))
	}
}

func TestBuildSenders_SendGrid_MissingFrom(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider:  "twilio-sendgrid",
		SendGridAPIKey: "SG.test-key",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Errorf("expected 0 senders when from is missing, got %d", len(senders))
	}
}

func TestBuildSenders_SMTP(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider: "smtp",
		SMTPHost:      "smtp.gmail.com",
		SMTPPort:      "587",
		SMTPUsername:  "user@gmail.com",
		SMTPPassword:  "app-password",
		EmailFrom:     "user@gmail.com",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 1 {
		t.Fatalf("expected 1 sender, got %d", len(senders))
	}
	if senders[0].Name() != "email" {
		t.Errorf("expected sender name=email, got: %s", senders[0].Name())
	}
}

func TestBuildSenders_SMTP_MissingHost(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider: "smtp",
		EmailFrom:     "user@gmail.com",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Errorf("expected 0 senders when SMTP host is missing, got %d", len(senders))
	}
}

func TestBuildSenders_SMTP_MissingFrom(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider: "smtp",
		SMTPHost:      "smtp.gmail.com",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Errorf("expected 0 senders when from is missing, got %d", len(senders))
	}
}

func TestBuildSenders_UnknownProvider(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider: "mailchimp",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Errorf("expected 0 senders for unknown provider, got %d", len(senders))
	}
}

func TestBuildSenders_SendGrid_CRLFInFrom(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider:  "twilio-sendgrid",
		SendGridAPIKey: "SG.test-key",
		EmailFrom:      "evil@example.com\r\nBcc: victim@example.com",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Errorf("expected 0 senders when from contains CRLF, got %d", len(senders))
	}
}

func TestBuildSenders_SMTP_CRLFInFrom(t *testing.T) {
	t.Parallel()
	cfg := Config{
		EmailProvider: "smtp",
		SMTPHost:      "smtp.gmail.com",
		SMTPPort:      "587",
		EmailFrom:     "evil@example.com\nBcc: victim@example.com",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Errorf("expected 0 senders when from contains newline, got %d", len(senders))
	}
}

func TestLogChannelSummary_NoSenders(t *testing.T) {
	t.Parallel()
	// Should not panic.
	LogChannelSummary(nil)
	LogChannelSummary([]Sender{})
}
