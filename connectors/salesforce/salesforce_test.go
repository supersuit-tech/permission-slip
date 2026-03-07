package salesforce

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSalesforceConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "salesforce" {
		t.Errorf("expected ID 'salesforce', got %q", c.ID())
	}
}

func TestSalesforceConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"salesforce.create_record",
		"salesforce.update_record",
		"salesforce.query",
		"salesforce.create_task",
		"salesforce.add_note",
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

func TestSalesforceConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid credentials",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "instance_url": "https://myorg.salesforce.com"}),
			wantErr: false,
		},
		{
			name:    "missing access_token",
			creds:   connectors.NewCredentials(map[string]string{"instance_url": "https://myorg.salesforce.com"}),
			wantErr: true,
		},
		{
			name:    "missing instance_url",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok"}),
			wantErr: true,
		},
		{
			name:    "empty access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "", "instance_url": "https://myorg.salesforce.com"}),
			wantErr: true,
		},
		{
			name:    "empty instance_url",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "instance_url": ""}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
		{
			name:    "http instance_url rejected",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "instance_url": "http://myorg.salesforce.com"}),
			wantErr: true,
		},
		{
			name:    "non-salesforce domain rejected",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "instance_url": "https://evil.example.com"}),
			wantErr: true,
		},
		{
			name:    "valid force.com domain",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "instance_url": "https://myorg.my.force.com"}),
			wantErr: false,
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

func TestSalesforceConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "salesforce" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "salesforce")
	}
	if m.Name != "Salesforce" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Salesforce")
	}
	if len(m.Actions) != 5 {
		t.Fatalf("Manifest().Actions has %d items, want 5", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"salesforce.create_record",
		"salesforce.update_record",
		"salesforce.query",
		"salesforce.create_task",
		"salesforce.add_note",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "salesforce" {
		t.Errorf("credential service = %q, want %q", cred.Service, "salesforce")
	}
	if cred.AuthType != "oauth2" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "oauth2")
	}
	if cred.OAuthProvider != "salesforce" {
		t.Errorf("credential oauth_provider = %q, want %q", cred.OAuthProvider, "salesforce")
	}
	if len(cred.OAuthScopes) == 0 {
		t.Error("credential oauth_scopes is empty, want at least one scope")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestSalesforceConnector_ActionsMatchManifest(t *testing.T) {
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

func TestSalesforceConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*SalesforceConnector)(nil)
	var _ connectors.ManifestProvider = (*SalesforceConnector)(nil)
}

func TestCheckResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		header     http.Header
		body       []byte
		wantNil    bool
		checkError func(t *testing.T, err error)
	}{
		{
			name:       "200 OK",
			statusCode: 200,
			body:       nil,
			wantNil:    true,
		},
		{
			name:       "204 No Content",
			statusCode: 204,
			body:       nil,
			wantNil:    true,
		},
		{
			name:       "401 Unauthorized",
			statusCode: 401,
			body:       []byte(`[{"errorCode":"INVALID_SESSION_ID","message":"Session expired"}]`),
			checkError: func(t *testing.T, err error) {
				if !connectors.IsAuthError(err) {
					t.Errorf("expected AuthError, got %T", err)
				}
			},
		},
		{
			name:       "403 Forbidden",
			statusCode: 403,
			body:       []byte(`[{"errorCode":"FORBIDDEN","message":"No access"}]`),
			checkError: func(t *testing.T, err error) {
				if !connectors.IsAuthError(err) {
					t.Errorf("expected AuthError for 403, got %T", err)
				}
			},
		},
		{
			name:       "429 Too Many Requests",
			statusCode: 429,
			header:     http.Header{"Retry-After": []string{"120"}},
			body:       []byte(`[]`),
			checkError: func(t *testing.T, err error) {
				if !connectors.IsRateLimitError(err) {
					t.Errorf("expected RateLimitError for 429, got %T", err)
				}
			},
		},
		{
			name:       "REQUEST_LIMIT_EXCEEDED error code",
			statusCode: 403,
			body:       []byte(`[{"errorCode":"REQUEST_LIMIT_EXCEEDED","message":"API limit hit"}]`),
			checkError: func(t *testing.T, err error) {
				if !connectors.IsRateLimitError(err) {
					t.Errorf("expected RateLimitError for REQUEST_LIMIT_EXCEEDED, got %T", err)
				}
			},
		},
		{
			name:       "MALFORMED_QUERY error code",
			statusCode: 400,
			body:       []byte(`[{"errorCode":"MALFORMED_QUERY","message":"bad query"}]`),
			checkError: func(t *testing.T, err error) {
				if !connectors.IsValidationError(err) {
					t.Errorf("expected ValidationError for MALFORMED_QUERY, got %T", err)
				}
			},
		},
		{
			name:       "500 Internal Server Error",
			statusCode: 500,
			body:       []byte(`[{"errorCode":"UNKNOWN","message":"something broke"}]`),
			checkError: func(t *testing.T, err error) {
				if !connectors.IsExternalError(err) {
					t.Errorf("expected ExternalError for 500, got %T", err)
				}
			},
		},
		{
			name:       "non-JSON error body",
			statusCode: 502,
			body:       []byte(`Bad Gateway`),
			checkError: func(t *testing.T, err error) {
				if !connectors.IsExternalError(err) {
					t.Errorf("expected ExternalError for 502 with non-JSON body, got %T", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			header := tt.header
			if header == nil {
				header = http.Header{}
			}
			err := checkResponse(tt.statusCode, header, tt.body)
			if tt.wantNil {
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			tt.checkError(t, err)
		})
	}
}

func TestValidateRecordID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id      string
		wantErr bool
	}{
		{"001xx0000000001", false},    // 15-char ID
		{"001xx0000000001AAA", false}, // 18-char ID
		{"abc", true},                // too short
		{"", true},                   // empty
		{"001xx000000000!", true},     // invalid character
		{"001xx00000000011234567890", true}, // too long
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			t.Parallel()
			err := validateRecordID(tt.id, "test_field")
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRecordID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestValidateInstanceURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url     string
		wantErr bool
	}{
		{"https://myorg.salesforce.com", false},
		{"https://myorg.my.salesforce.com", false},
		{"https://myorg.my.force.com", false},
		{"https://na1.salesforce.com", false},
		{"http://myorg.salesforce.com", true},           // not HTTPS
		{"https://evil.example.com", true},               // wrong domain
		{"https://salesforce.com.evil.com", true},         // subdomain trick
		{"ftp://myorg.salesforce.com", true},              // wrong scheme
		{"not-a-url", true},                               // invalid URL
		{"https://", true},                                // no host
		{"https://evil.com/salesforce.com", true},         // path, not host
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			err := validateInstanceURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateInstanceURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestRecordURL(t *testing.T) {
	t.Parallel()

	creds := connectors.NewCredentials(map[string]string{
		"access_token": "tok",
		"instance_url": "https://myorg.salesforce.com",
	})
	got := recordURL(creds, "001xx0000000001")
	want := "https://myorg.salesforce.com/001xx0000000001"
	if got != want {
		t.Errorf("recordURL() = %q, want %q", got, want)
	}

	// No instance_url → empty string.
	emptyCreds := connectors.NewCredentials(map[string]string{"access_token": "tok"})
	if url := recordURL(emptyCreds, "001xx0000000001"); url != "" {
		t.Errorf("expected empty url for missing instance_url, got %q", url)
	}
}
