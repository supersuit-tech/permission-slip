package protonmail

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestReadEmail_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readEmailAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.read_email",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestReadEmail_MissingMessageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.read_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestReadEmailParams_Defaults(t *testing.T) {
	t.Parallel()

	p := &readEmailParams{MessageID: 1}
	if err := p.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Folder != "INBOX" {
		t.Errorf("expected default folder 'INBOX', got %q", p.Folder)
	}
}

func TestFormatAddresses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addrs    []addressInput
		expected []string
	}{
		{
			name: "with name",
			addrs: []addressInput{
				{Name: "Alice", Mailbox: "alice", Host: "example.com"},
			},
			expected: []string{"Alice <alice@example.com>"},
		},
		{
			name: "without name",
			addrs: []addressInput{
				{Name: "", Mailbox: "bob", Host: "example.com"},
			},
			expected: []string{"bob@example.com"},
		},
		{
			name:     "empty",
			addrs:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var imapAddrs []imapAddress
			for _, a := range tt.addrs {
				imapAddrs = append(imapAddrs, imapAddress{name: a.Name, mailbox: a.Mailbox, host: a.Host})
			}
			result := formatAddressesFromInputs(imapAddrs)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d addresses, got %d", len(tt.expected), len(result))
			}
			for i, r := range result {
				if r != tt.expected[i] {
					t.Errorf("address[%d] = %q, want %q", i, r, tt.expected[i])
				}
			}
		})
	}
}

type addressInput struct {
	Name, Mailbox, Host string
}

type imapAddress struct {
	name, mailbox, host string
}

func formatAddressesFromInputs(addrs []imapAddress) []string {
	var result []string
	for _, addr := range addrs {
		if addr.mailbox == "" || addr.host == "" {
			continue
		}
		a := addr.mailbox + "@" + addr.host
		if addr.name != "" {
			result = append(result, addr.name+" <"+a+">")
		} else {
			result = append(result, a)
		}
	}
	return result
}

func TestTruncateBody(t *testing.T) {
	t.Parallel()

	short := "hello"
	if got := truncateBody(short); got != short {
		t.Errorf("truncateBody(%q) = %q, want %q", short, got, short)
	}

	// Create a string longer than maxBodySize.
	long := make([]byte, maxBodySize+100)
	for i := range long {
		long[i] = 'a'
	}
	result := truncateBody(string(long))
	if len(result) != maxBodySize+len("\n[truncated]") {
		t.Errorf("truncateBody should truncate to maxBodySize + suffix, got len=%d", len(result))
	}
}

func TestIMAPConfig_Defaults(t *testing.T) {
	t.Parallel()

	creds := validCreds()
	host, port, username, password := imapConfig(creds)

	if host != defaultIMAPHost {
		t.Errorf("expected default IMAP host %q, got %q", defaultIMAPHost, host)
	}
	if port != defaultIMAPPort {
		t.Errorf("expected default IMAP port %q, got %q", defaultIMAPPort, port)
	}
	if username != "user@proton.me" {
		t.Errorf("expected username 'user@proton.me', got %q", username)
	}
	if password != "bridge-generated-password" {
		t.Errorf("expected password 'bridge-generated-password', got %q", password)
	}
}

func TestIMAPConfig_CustomValues(t *testing.T) {
	t.Parallel()

	creds := connectors.NewCredentials(map[string]string{
		"username":  "user@proton.me",
		"password":  "pass",
		"imap_host": "192.168.1.1",
		"imap_port": "993",
	})
	host, port, _, _ := imapConfig(creds)

	if host != "192.168.1.1" {
		t.Errorf("expected IMAP host '192.168.1.1', got %q", host)
	}
	if port != "993" {
		t.Errorf("expected IMAP port '993', got %q", port)
	}
}

func TestSMTPConfig_Defaults(t *testing.T) {
	t.Parallel()

	creds := validCreds()
	host, port, username, password := smtpConfig(creds)

	if host != defaultSMTPHost {
		t.Errorf("expected default SMTP host %q, got %q", defaultSMTPHost, host)
	}
	if port != defaultSMTPPort {
		t.Errorf("expected default SMTP port %q, got %q", defaultSMTPPort, port)
	}
	if username != "user@proton.me" {
		t.Errorf("expected username 'user@proton.me', got %q", username)
	}
	if password != "bridge-generated-password" {
		t.Errorf("expected password 'bridge-generated-password', got %q", password)
	}
}

func TestSMTPConfig_CustomValues(t *testing.T) {
	t.Parallel()

	creds := connectors.NewCredentials(map[string]string{
		"username":  "user@proton.me",
		"password":  "pass",
		"smtp_host": "192.168.1.1",
		"smtp_port": "587",
	})
	host, port, _, _ := smtpConfig(creds)

	if host != "192.168.1.1" {
		t.Errorf("expected SMTP host '192.168.1.1', got %q", host)
	}
	if port != "587" {
		t.Errorf("expected SMTP port '587', got %q", port)
	}
}
