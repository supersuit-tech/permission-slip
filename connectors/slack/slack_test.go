package slack

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
		"slack.send_dm",
		"slack.update_message",
		"slack.delete_message",
		"slack.list_users",
		"slack.search_messages",
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
			name:    "valid bot_token",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "xoxb-1234567890-abcdef"}),
			wantErr: false,
		},
		{
			name:    "valid access_token (OAuth)",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "xoxb-oauth-token-value"}),
			wantErr: false,
		},
		{
			name:    "access_token preferred over bot_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "oauth-tok", "bot_token": "xoxb-bot-tok"}),
			wantErr: false,
		},
		{
			name:    "missing both tokens",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty bot_token",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": ""}),
			wantErr: true,
		},
		{
			name:    "empty access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
			wantErr: true,
		},
		{
			name:    "wrong prefix xoxp",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "xoxp-user-token"}),
			wantErr: true,
		},
		{
			name:    "wrong prefix xoxa",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "xoxa-app-token"}),
			wantErr: true,
		},
		{
			name:    "wrong prefix bearer",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "bearer-123"}),
			wantErr: true,
		},
		{
			name:    "wrong prefix sk",
			creds:   connectors.NewCredentials(map[string]string{"bot_token": "sk-123"}),
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
	if len(m.Actions) != 15 {
		t.Fatalf("Manifest().Actions has %d items, want 15", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"slack.send_message", "slack.create_channel", "slack.list_channels",
		"slack.read_channel_messages", "slack.read_thread",
		"slack.schedule_message", "slack.set_topic", "slack.invite_to_channel",
		"slack.upload_file", "slack.add_reaction",
		"slack.send_dm", "slack.update_message", "slack.delete_message",
		"slack.list_users", "slack.search_messages",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}

	// First credential: OAuth2 (primary / recommended)
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

	// Second credential: bot token (legacy / alternative)
	botCred := m.RequiredCredentials[1]
	if botCred.Service != "slack_bot" {
		t.Errorf("bot credential service = %q, want %q", botCred.Service, "slack_bot")
	}
	if botCred.AuthType != "custom" {
		t.Errorf("bot credential auth_type = %q, want %q", botCred.AuthType, "custom")
	}
	if botCred.InstructionsURL == "" {
		t.Error("bot credential instructions_url is empty, want a URL")
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
				if !connectors.IsAuthError(err) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				}
			case "rate_limit":
				if !connectors.IsRateLimitError(err) {
					t.Errorf("expected RateLimitError, got %T: %v", err, err)
				}
			case "external":
				if !connectors.IsExternalError(err) {
					t.Errorf("expected ExternalError, got %T: %v", err, err)
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
				if !connectors.IsAuthError(err) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				}
			case "rate_limit":
				var rle *connectors.RateLimitError
				if !connectors.AsRateLimitError(err, &rle) {
					t.Errorf("expected RateLimitError, got %T: %v", err, err)
				} else if rle.RetryAfter != 60*time.Second {
					t.Errorf("RetryAfter = %v, want 60s", rle.RetryAfter)
				}
			case "external":
				if !connectors.IsExternalError(err) {
					t.Errorf("expected ExternalError, got %T: %v", err, err)
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
