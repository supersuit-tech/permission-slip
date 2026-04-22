package slack

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSlackConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "slack" {
		t.Errorf("expected ID 'slack', got %q", c.ID())
	}
}

func TestSlackConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"slack.send_message",
		"slack.create_channel",
		"slack.list_channels",
		"slack.read_channel_messages",
		"slack.read_thread",
		"slack.schedule_message",
		"slack.set_topic",
		"slack.invite_to_channel",
		"slack.upload_file",
		"slack.add_reaction",
		"slack.remove_reaction",
		"slack.send_dm",
		"slack.update_message",
		"slack.delete_message",
		"slack.list_users",
		"slack.search_messages",
		"slack.archive_channel",
		"slack.rename_channel",
		"slack.remove_from_channel",
		"slack.get_user_profile",
		"slack.pin_message",
		"slack.unpin_message",
		"slack.list_unread",
		"slack.mark_read",
	}
	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("expected action %q to be registered", name)
		}
	}

	if len(actions) != len(expected) {
		t.Errorf("expected %d actions, got %d", len(expected), len(actions))
	}
}

func TestSlackConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid access_token (OAuth user)",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "xoxp-1234567890-abcdef"}),
			wantErr: false,
		},
		{
			name:    "missing token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("ValidateCredentials() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestSlackConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "slack" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "slack")
	}
	if m.Name != "Slack" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Slack")
	}
	if len(m.Actions) != 24 {
		t.Fatalf("Manifest().Actions has %d items, want 24", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"slack.send_message", "slack.create_channel", "slack.list_channels",
		"slack.read_channel_messages", "slack.read_thread",
		"slack.schedule_message", "slack.set_topic", "slack.invite_to_channel",
		"slack.upload_file", "slack.add_reaction", "slack.remove_reaction",
		"slack.send_dm", "slack.update_message", "slack.delete_message",
		"slack.list_users", "slack.search_messages",
		"slack.archive_channel", "slack.rename_channel", "slack.remove_from_channel",
		"slack.get_user_profile", "slack.pin_message", "slack.unpin_message",
		"slack.list_unread", "slack.mark_read",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}

	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "slack" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "slack")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "slack" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "slack")
	}
	if len(oauthCred.OAuthScopes) == 0 {
		t.Error("oauth credential oauth_scopes is empty, want at least one scope")
	}
	if len(oauthCred.OAuthScopes) != len(OAuthScopes) {
		t.Errorf("oauth credential oauth_scopes has %d entries, want %d (OAuthScopes var)", len(oauthCred.OAuthScopes), len(OAuthScopes))
	}

	// Validate the manifest passes validation.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestSlackConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()

	manifestTypes := make(map[string]bool, len(manifest.Actions))
	for _, a := range manifest.Actions {
		manifestTypes[a.ActionType] = true
	}

	for actionType := range actions {
		if !manifestTypes[actionType] {
			t.Errorf("Actions() has %q but Manifest() does not", actionType)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("Manifest() has %q but Actions() does not", a.ActionType)
		}
	}
}

func TestSlackConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*SlackConnector)(nil)
	var _ connectors.ManifestProvider = (*SlackConnector)(nil)
}

func TestValidateUserID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{name: "valid U prefix", userID: "U01234567", wantErr: false},
		{name: "valid W prefix", userID: "W01234567", wantErr: false},
		{name: "empty", userID: "", wantErr: true},
		{name: "single char", userID: "U", wantErr: true},
		{name: "username instead of ID", userID: "john.doe", wantErr: true},
		{name: "email instead of ID", userID: "john@example.com", wantErr: true},
		{name: "channel ID", userID: "C01234567", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateUserID(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUserID(%q) error = %v, wantErr %v", tt.userID, err, tt.wantErr)
			}
		})
	}
}

func TestCheckHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantType   string // "auth", "rate_limit", "external"
	}{
		{
			name:       "401 with Slack error",
			statusCode: 401,
			body:       `{"ok":false,"error":"invalid_auth"}`,
			wantType:   "auth",
		},
		{
			name:       "401 without body",
			statusCode: 401,
			body:       ``,
			wantType:   "auth",
		},
		{
			name:       "403 permission denied",
			statusCode: 403,
			body:       `{"ok":false,"error":"missing_scope"}`,
			wantType:   "auth",
		},
		{
			name:       "429 rate limited",
			statusCode: 429,
			body:       `{"ok":false,"error":"ratelimited"}`,
			wantType:   "rate_limit",
		},
		{
			name:       "500 server error",
			statusCode: 500,
			body:       `internal server error`,
			wantType:   "external",
		},
		{
			name:       "503 service unavailable with JSON",
			statusCode: 503,
			body:       `{"ok":false,"error":"service_unavailable"}`,
			wantType:   "external",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkHTTPStatus(tt.statusCode, http.Header{}, []byte(tt.body))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			switch tt.wantType {
			case "auth":
				var ae *connectors.AuthError
				if !errors.As(err, &ae) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				} else if ae.Message == "" {
					t.Error("AuthError.Message is empty")
				}
			case "rate_limit":
				var rle *connectors.RateLimitError
				if !connectors.AsRateLimitError(err, &rle) {
					t.Errorf("expected RateLimitError, got %T: %v", err, err)
				} else if rle.RetryAfter != defaultRetryAfter {
					t.Errorf("RetryAfter = %v, want %v (default)", rle.RetryAfter, defaultRetryAfter)
				}
			case "external":
				var ee *connectors.ExternalError
				if !errors.As(err, &ee) {
					t.Errorf("expected ExternalError, got %T: %v", err, err)
				} else if ee.StatusCode != tt.statusCode {
					t.Errorf("StatusCode = %d, want %d", ee.StatusCode, tt.statusCode)
				}
			}
		})
	}
}

func TestDoPost_HTTPErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantType   string
	}{
		{
			name:       "401 returns AuthError",
			statusCode: 401,
			body:       `{"ok":false,"error":"invalid_auth"}`,
			wantType:   "auth",
		},
		{
			name:       "429 returns RateLimitError",
			statusCode: 429,
			body:       `{"ok":false,"error":"ratelimited"}`,
			wantType:   "rate_limit",
		},
		{
			name:       "500 returns ExternalError",
			statusCode: 500,
			body:       `not json`,
			wantType:   "external",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode == http.StatusTooManyRequests {
					w.Header().Set("Retry-After", "60")
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := newForTest(srv.Client(), srv.URL)
			var dest slackResponse
			err := c.doPost(t.Context(), "test.method", validCreds(), map[string]string{}, &dest)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			switch tt.wantType {
			case "auth":
				var ae *connectors.AuthError
				if !errors.As(err, &ae) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				} else if ae.Message == "" {
					t.Error("AuthError.Message is empty")
				}
			case "rate_limit":
				var rle *connectors.RateLimitError
				if !connectors.AsRateLimitError(err, &rle) {
					t.Errorf("expected RateLimitError, got %T: %v", err, err)
				} else if rle.RetryAfter != 60*time.Second {
					t.Errorf("RetryAfter = %v, want 60s", rle.RetryAfter)
				}
			case "external":
				var ee *connectors.ExternalError
				if !errors.As(err, &ee) {
					t.Errorf("expected ExternalError, got %T: %v", err, err)
				} else if ee.StatusCode != tt.statusCode {
					t.Errorf("StatusCode = %d, want %d", ee.StatusCode, tt.statusCode)
				}
			}
		})
	}
}

func TestValidateMessageTS(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ts      string
		wantErr bool
	}{
		{name: "valid ts", ts: "1234567890.123456", wantErr: false},
		{name: "empty", ts: "", wantErr: true},
		{name: "no dot", ts: "1234567890123456", wantErr: true},
		{name: "letters", ts: "not-a-timestamp", wantErr: true},
		{name: "mixed", ts: "123abc.456", wantErr: true},
		{name: "multiple dots", ts: "1.2.3", wantErr: true},
		{name: "leading dot", ts: ".123456", wantErr: true},
		{name: "trailing dot", ts: "123456.", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateMessageTS(tt.ts)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMessageTS(%q) error = %v, wantErr %v", tt.ts, err, tt.wantErr)
			}
		})
	}
}

func TestGrantedScopes(t *testing.T) {
	t.Parallel()

	h := http.Header{}
	if got := grantedScopes(h); got != nil {
		t.Errorf("missing header should return nil, got %v", got)
	}

	h.Set("X-OAuth-Scopes", "im:read, users:read.email ,channels:read,")
	got := grantedScopes(h)
	for _, want := range []string{"im:read", "users:read.email", "channels:read"} {
		if !got[want] {
			t.Errorf("expected scope %q in %v", want, got)
		}
	}
	if got[""] {
		t.Error("empty scope entries must not be kept")
	}
}

func TestMissingScopes(t *testing.T) {
	t.Parallel()

	if got := missingScopes(nil, "im:read"); got != nil {
		t.Errorf("nil granted header must return nil (no false positives), got %v", got)
	}
	granted := map[string]bool{"im:read": true, "users:read.email": true}
	if got := missingScopes(granted, "im:read"); len(got) != 0 {
		t.Errorf("expected no missing, got %v", got)
	}
	got := missingScopes(granted, "im:read", "mpim:read", "groups:read")
	if len(got) != 2 || got[0] != "mpim:read" || got[1] != "groups:read" {
		t.Errorf("expected [mpim:read groups:read], got %v", got)
	}
}

func TestRequiredPrivateTypeScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		types string
		want  []string
	}{
		{"", nil},
		{"public_channel", nil},
		{"im", []string{"im:read"}},
		{"mpim", []string{"mpim:read"}},
		{"private_channel", []string{"groups:read"}},
		{"im,mpim,private_channel", []string{"im:read", "mpim:read", "groups:read"}},
		{"im, im ,im", []string{"im:read"}}, // dedupe
	}
	for _, tt := range tests {
		got := requiredPrivateTypeScopes(tt.types)
		if len(got) != len(tt.want) {
			t.Errorf("requiredPrivateTypeScopes(%q) = %v, want %v", tt.types, got, tt.want)
			continue
		}
		for i, s := range tt.want {
			if got[i] != s {
				t.Errorf("requiredPrivateTypeScopes(%q)[%d] = %q, want %q", tt.types, i, got[i], s)
			}
		}
	}
}
