package connectors

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestParseManifest_Valid(t *testing.T) {
	input := `{
		"id": "jira",
		"name": "Jira",
		"description": "Jira issue tracking",
		"actions": [
			{
				"action_type": "jira.create_issue",
				"name": "Create Issue",
				"description": "Create a Jira issue",
				"risk_level": "low",
				"parameters_schema": {}
			}
		],
		"required_credentials": [
			{"service": "jira", "auth_type": "api_key"}
		]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID != "jira" {
		t.Errorf("ID = %q, want %q", m.ID, "jira")
	}
	if m.Name != "Jira" {
		t.Errorf("Name = %q, want %q", m.Name, "Jira")
	}
	if len(m.Actions) != 1 {
		t.Fatalf("len(Actions) = %d, want 1", len(m.Actions))
	}
	if m.Actions[0].ActionType != "jira.create_issue" {
		t.Errorf("ActionType = %q, want %q", m.Actions[0].ActionType, "jira.create_issue")
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("len(RequiredCredentials) = %d, want 1", len(m.RequiredCredentials))
	}
}

func TestParseManifest_MultipleActions(t *testing.T) {
	input := `{
		"id": "myservice",
		"name": "My Service",
		"description": "A test service",
		"actions": [
			{"action_type": "myservice.action_a", "name": "Action A", "risk_level": "low"},
			{"action_type": "myservice.action_b", "name": "Action B", "risk_level": "high"}
		],
		"required_credentials": [
			{"service": "myservice", "auth_type": "custom"}
		]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Actions) != 2 {
		t.Fatalf("len(Actions) = %d, want 2", len(m.Actions))
	}
}

func TestParseManifest_NoCredentials(t *testing.T) {
	input := `{
		"id": "noop",
		"name": "No-Op",
		"actions": [{"action_type": "noop.ping", "name": "Ping"}]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.RequiredCredentials) != 0 {
		t.Errorf("len(RequiredCredentials) = %d, want 0", len(m.RequiredCredentials))
	}
}

func TestParseManifest_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty id", `{"id":"","name":"X","actions":[{"action_type":"x.a","name":"A"}]}`},
		{"invalid id chars", `{"id":"Foo Bar","name":"X","actions":[{"action_type":"Foo Bar.a","name":"A"}]}`},
		{"missing name", `{"id":"x","actions":[{"action_type":"x.a","name":"A"}]}`},
		{"no actions", `{"id":"x","name":"X","actions":[]}`},
		{"missing action_type", `{"id":"x","name":"X","actions":[{"name":"A"}]}`},
		{"wrong action prefix", `{"id":"x","name":"X","actions":[{"action_type":"y.a","name":"A"}]}`},
		{"duplicate action_type", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"},{"action_type":"x.a","name":"B"}]}`},
		{"missing action name", `{"id":"x","name":"X","actions":[{"action_type":"x.a"}]}`},
		{"invalid risk_level", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A","risk_level":"extreme"}]}`},
		{"duplicate cred service+auth_type", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"api_key"},{"service":"s","auth_type":"api_key"}]}`},
		{"missing cred service", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"auth_type":"api_key"}]}`},
		{"missing auth_type", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s"}]}`},
		{"invalid auth_type", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"magic"}]}`},
		{"oauth2 missing oauth_provider", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"oauth2"}]}`},
		{"invalid JSON", `{not json`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tt.input))
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// ── OAuth Validation ─────────────────────────────────────────────────────

func TestParseManifest_MultipleAuthTypesForSameService(t *testing.T) {
	input := `{
		"id": "pagerduty",
		"name": "PagerDuty",
		"actions": [{"action_type": "pagerduty.create_incident", "name": "Create Incident"}],
		"required_credentials": [
			{"service": "pagerduty", "auth_type": "oauth2", "oauth_provider": "pagerduty"},
			{"service": "pagerduty", "auth_type": "api_key"}
		]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("len(RequiredCredentials) = %d, want 2", len(m.RequiredCredentials))
	}
	if m.RequiredCredentials[0].AuthType != "oauth2" {
		t.Errorf("first credential auth_type = %q, want %q", m.RequiredCredentials[0].AuthType, "oauth2")
	}
	if m.RequiredCredentials[1].AuthType != "api_key" {
		t.Errorf("second credential auth_type = %q, want %q", m.RequiredCredentials[1].AuthType, "api_key")
	}
}

func TestParseManifest_OAuth2Credential(t *testing.T) {
	input := `{
		"id": "google",
		"name": "Google",
		"actions": [{"action_type": "google.send_email", "name": "Send Email"}],
		"required_credentials": [
			{"service": "google", "auth_type": "oauth2", "oauth_provider": "google", "oauth_scopes": ["https://www.googleapis.com/auth/gmail.send"]}
		]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("len(RequiredCredentials) = %d, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.AuthType != "oauth2" {
		t.Errorf("AuthType = %q, want %q", cred.AuthType, "oauth2")
	}
	if cred.OAuthProvider != "google" {
		t.Errorf("OAuthProvider = %q, want %q", cred.OAuthProvider, "google")
	}
	if len(cred.OAuthScopes) != 1 || cred.OAuthScopes[0] != "https://www.googleapis.com/auth/gmail.send" {
		t.Errorf("OAuthScopes = %v, want [https://www.googleapis.com/auth/gmail.send]", cred.OAuthScopes)
	}
}

func TestParseManifest_OAuth2ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"oauth2 without oauth_provider", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"oauth2"}]}`},
		{"non-oauth2 with oauth_provider", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"api_key","oauth_provider":"google"}]}`},
		{"non-oauth2 with oauth_scopes", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"api_key","oauth_scopes":["scope1"]}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tt.input))
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseManifest_OAuthProviders(t *testing.T) {
	input := `{
		"id": "salesforce",
		"name": "Salesforce",
		"actions": [{"action_type": "salesforce.query", "name": "Query"}],
		"required_credentials": [
			{"service": "salesforce", "auth_type": "oauth2", "oauth_provider": "salesforce"}
		],
		"oauth_providers": [
			{
				"id": "salesforce",
				"authorize_url": "https://login.salesforce.com/services/oauth2/authorize",
				"token_url": "https://login.salesforce.com/services/oauth2/token",
				"scopes": ["api", "refresh_token"]
			}
		]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.OAuthProviders) != 1 {
		t.Fatalf("len(OAuthProviders) = %d, want 1", len(m.OAuthProviders))
	}
	p := m.OAuthProviders[0]
	if p.ID != "salesforce" {
		t.Errorf("OAuthProvider ID = %q, want %q", p.ID, "salesforce")
	}
	if p.AuthorizeURL != "https://login.salesforce.com/services/oauth2/authorize" {
		t.Errorf("OAuthProvider AuthorizeURL = %q", p.AuthorizeURL)
	}
	if p.TokenURL != "https://login.salesforce.com/services/oauth2/token" {
		t.Errorf("OAuthProvider TokenURL = %q", p.TokenURL)
	}
	if len(p.Scopes) != 2 {
		t.Errorf("OAuthProvider Scopes = %v, want [api refresh_token]", p.Scopes)
	}
}

func TestParseManifest_OAuthProviderCrossReference(t *testing.T) {
	// Built-in providers (google, microsoft) should be accepted without declaration.
	t.Run("built-in provider accepted", func(t *testing.T) {
		input := `{
			"id": "x",
			"name": "X",
			"actions": [{"action_type": "x.a", "name": "A"}],
			"required_credentials": [
				{"service": "s", "auth_type": "oauth2", "oauth_provider": "microsoft"}
			]
		}`
		_, err := ParseManifest([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Custom provider declared in oauth_providers should be accepted.
	t.Run("declared custom provider accepted", func(t *testing.T) {
		input := `{
			"id": "x",
			"name": "X",
			"actions": [{"action_type": "x.a", "name": "A"}],
			"required_credentials": [
				{"service": "s", "auth_type": "oauth2", "oauth_provider": "hubspot"}
			],
			"oauth_providers": [
				{"id": "hubspot", "authorize_url": "https://app.hubspot.com/oauth/authorize", "token_url": "https://api.hubapi.com/oauth/v1/token"}
			]
		}`
		_, err := ParseManifest([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Unknown provider not declared should fail.
	t.Run("undeclared custom provider rejected", func(t *testing.T) {
		input := `{
			"id": "x",
			"name": "X",
			"actions": [{"action_type": "x.a", "name": "A"}],
			"required_credentials": [
				{"service": "s", "auth_type": "oauth2", "oauth_provider": "unknown-custom-provider"}
			]
		}`
		_, err := ParseManifest([]byte(input))
		if err == nil {
			t.Error("expected error for undeclared oauth_provider, got nil")
		}
	})
}

func TestParseManifest_OAuthProviderValidationErrors(t *testing.T) {
	base := `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"oauth_providers":[%s]}`

	tests := []struct {
		name     string
		provider string
	}{
		{"missing provider id", `{"authorize_url":"https://example.com/auth","token_url":"https://example.com/token"}`},
		{"missing authorize_url", `{"id":"p","token_url":"https://example.com/token"}`},
		{"missing token_url", `{"id":"p","authorize_url":"https://example.com/auth"}`},
		{"non-https authorize_url", `{"id":"p","authorize_url":"http://example.com/auth","token_url":"https://example.com/token"}`},
		{"non-https token_url", `{"id":"p","authorize_url":"https://example.com/auth","token_url":"http://example.com/token"}`},
		{"duplicate provider id", `{"id":"p","authorize_url":"https://a.com/auth","token_url":"https://a.com/token"},{"id":"p","authorize_url":"https://b.com/auth","token_url":"https://b.com/token"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf(base, tt.provider)
			_, err := ParseManifest([]byte(input))
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseManifest_InstructionsURL(t *testing.T) {
	input := `{
		"id": "test",
		"name": "Test",
		"actions": [{"action_type": "test.ping", "name": "Ping"}],
		"required_credentials": [
			{"service": "test", "auth_type": "api_key", "instructions_url": "https://docs.example.com/setup"}
		]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.RequiredCredentials[0].InstructionsURL != "https://docs.example.com/setup" {
		t.Errorf("InstructionsURL = %q, want %q", m.RequiredCredentials[0].InstructionsURL, "https://docs.example.com/setup")
	}
}

func TestParseManifest_InstructionsURLValidation(t *testing.T) {
	base := func(url string) string {
		return `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"api_key","instructions_url":"` + url + `"}]}`
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid https URL", base("https://docs.example.com/setup"), false},
		{"valid http URL", base("http://docs.example.com/setup"), false},
		{"empty URL (omitted)", `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"api_key"}]}`, false},
		{"ftp scheme rejected", base("ftp://example.com/file"), true},
		{"javascript scheme rejected", base("javascript:alert(1)"), true},
		{"no scheme rejected", base("example.com/setup"), true},
		{"empty host rejected", base("https://"), true},
		{"empty host with path rejected", base("https:///setup"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseManifest_InstructionsURLTooLong(t *testing.T) {
	// Build a URL that exceeds 2048 chars.
	longURL := "https://example.com/"
	for len(longURL) < 2049 {
		longURL += "a"
	}
	input := `{"id":"x","name":"X","actions":[{"action_type":"x.a","name":"A"}],"required_credentials":[{"service":"s","auth_type":"api_key","instructions_url":"` + longURL + `"}]}`
	_, err := ParseManifest([]byte(input))
	if err == nil {
		t.Error("expected error for URL exceeding max length, got nil")
	}
}

func TestParseManifest_ParametersSchemaPreserved(t *testing.T) {
	input := `{
		"id": "x",
		"name": "X",
		"actions": [{
			"action_type": "x.do",
			"name": "Do",
			"parameters_schema": {"type": "object", "properties": {"foo": {"type": "string"}}}
		}]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Actions[0].ParametersSchema == nil {
		t.Fatal("ParametersSchema is nil")
	}

	// Verify it round-trips as valid JSON.
	var schema map[string]interface{}
	if err := json.Unmarshal(m.Actions[0].ParametersSchema, &schema); err != nil {
		t.Fatalf("ParametersSchema is not valid JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Errorf("schema type = %v, want %q", schema["type"], "object")
	}
}

// ── Template Validation ─────────────────────────────────────────────────────

func TestParseManifest_WithTemplates(t *testing.T) {
	input := `{
		"id": "x",
		"name": "X",
		"actions": [{"action_type": "x.do", "name": "Do"}],
		"templates": [
			{
				"id": "tpl_x_do_default",
				"action_type": "x.do",
				"name": "Default config",
				"description": "Pre-filled for the common case",
				"parameters": {"key": "*"}
			}
		]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Templates) != 1 {
		t.Fatalf("len(Templates) = %d, want 1", len(m.Templates))
	}
	if m.Templates[0].ID != "tpl_x_do_default" {
		t.Errorf("Template ID = %q, want %q", m.Templates[0].ID, "tpl_x_do_default")
	}
	if m.Templates[0].ActionType != "x.do" {
		t.Errorf("Template ActionType = %q, want %q", m.Templates[0].ActionType, "x.do")
	}
	if m.Templates[0].Name != "Default config" {
		t.Errorf("Template Name = %q, want %q", m.Templates[0].Name, "Default config")
	}
	if m.Templates[0].Description != "Pre-filled for the common case" {
		t.Errorf("Template Description = %q", m.Templates[0].Description)
	}

	// Parameters should round-trip as valid JSON.
	var params map[string]interface{}
	if err := json.Unmarshal(m.Templates[0].Parameters, &params); err != nil {
		t.Fatalf("Template Parameters is not valid JSON: %v", err)
	}
}

func TestParseManifest_NoTemplates(t *testing.T) {
	input := `{
		"id": "x",
		"name": "X",
		"actions": [{"action_type": "x.do", "name": "Do"}]
	}`

	m, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Templates) != 0 {
		t.Errorf("len(Templates) = %d, want 0", len(m.Templates))
	}
}

func TestParseManifest_TemplateValidationErrors(t *testing.T) {
	base := `{"id":"x","name":"X","actions":[{"action_type":"x.do","name":"Do"}],"templates":[%s]}`

	tests := []struct {
		name     string
		template string
	}{
		{"missing template id", `{"action_type":"x.do","name":"T","parameters":{"k":"v"}}`},
		{"missing template action_type", `{"id":"tpl_x_1","name":"T","parameters":{"k":"v"}}`},
		{"template action_type not in actions", `{"id":"tpl_x_1","action_type":"x.other","name":"T","parameters":{"k":"v"}}`},
		{"missing template name", `{"id":"tpl_x_1","action_type":"x.do","parameters":{"k":"v"}}`},
		{"missing template parameters", `{"id":"tpl_x_1","action_type":"x.do","name":"T"}`},
		{"duplicate template id", `{"id":"tpl_x_1","action_type":"x.do","name":"A","parameters":{"k":"v"}},{"id":"tpl_x_1","action_type":"x.do","name":"B","parameters":{"k":"v"}}`},
		{"template id missing connector id", `{"id":"tpl_other_1","action_type":"x.do","name":"T","parameters":{"k":"v"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := fmt.Sprintf(base, tt.template)
			_, err := ParseManifest([]byte(input))
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestManifestValidation_ReservedAuthorizeParams(t *testing.T) {
	t.Parallel()

	// Base manifest with a valid OAuth provider. The %s placeholder is for
	// authorize_params JSON.
	base := `{
		"id": "test",
		"name": "Test",
		"actions": [{"action_type": "test.do", "name": "Do", "risk_level": "low"}],
		"required_credentials": [
			{"service": "test_oauth", "auth_type": "oauth2", "oauth_provider": "test"}
		],
		"oauth_providers": [{
			"id": "test",
			"authorize_url": "https://auth.example.com/authorize",
			"token_url": "https://auth.example.com/token",
			"authorize_params": {%s}
		}]
	}`

	t.Run("valid_params_accepted", func(t *testing.T) {
		input := fmt.Sprintf(base, `"audience": "api.example.com", "prompt": "consent"`)
		_, err := ParseManifest([]byte(input))
		if err != nil {
			t.Fatalf("expected valid params to be accepted, got: %v", err)
		}
	})

	for _, reserved := range []string{"redirect_uri", "state", "client_id", "client_secret", "response_type", "code", "grant_type"} {
		t.Run("rejects_"+reserved, func(t *testing.T) {
			input := fmt.Sprintf(base, fmt.Sprintf(`"%s": "evil-value"`, reserved))
			_, err := ParseManifest([]byte(input))
			if err == nil {
				t.Errorf("expected error for reserved param %q, got nil", reserved)
			}
		})
	}
}

// ── x-ui Validation ─────────────────────────────────────────────────────

func TestParseManifest_ValidXUI(t *testing.T) {
	input := `{
		"id": "x",
		"name": "X",
		"actions": [{
			"action_type": "x.do",
			"name": "Do",
			"parameters_schema": {
				"type": "object",
				"x-ui": {
					"groups": [
						{"id": "main", "label": "Main Settings"},
						{"id": "advanced", "label": "Advanced", "collapsed": true}
					],
					"order": ["name", "email", "notify"]
				},
				"properties": {
					"name": {
						"type": "string",
						"x-ui": {"label": "Full Name", "placeholder": "John Doe", "group": "main", "widget": "text"}
					},
					"email": {
						"type": "string",
						"x-ui": {"label": "Email", "group": "main", "help_url": "https://example.com/help"}
					},
					"notify": {
						"type": "boolean",
						"x-ui": {"widget": "toggle", "group": "advanced", "visible_when": {"field": "email", "equals": "set"}}
					}
				}
			}
		}]
	}`

	_, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseManifest_XUIWithoutGroups(t *testing.T) {
	// x-ui on properties without root groups or order should be fine
	// as long as no group references are made.
	input := `{
		"id": "x",
		"name": "X",
		"actions": [{
			"action_type": "x.do",
			"name": "Do",
			"parameters_schema": {
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"x-ui": {"label": "Name", "widget": "text", "placeholder": "Enter name"}
					}
				}
			}
		}]
	}`

	_, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseManifest_NoXUI(t *testing.T) {
	// Schema without any x-ui should pass validation (backwards compatible).
	input := `{
		"id": "x",
		"name": "X",
		"actions": [{
			"action_type": "x.do",
			"name": "Do",
			"parameters_schema": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		}]
	}`

	_, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseManifest_XUIValidationErrors(t *testing.T) {
	base := func(schema string) string {
		return fmt.Sprintf(`{"id":"x","name":"X","actions":[{"action_type":"x.do","name":"Do","parameters_schema":%s}]}`, schema)
	}

	tests := []struct {
		name   string
		schema string
	}{
		{
			"invalid widget type",
			`{"type":"object","properties":{"f":{"type":"string","x-ui":{"widget":"slider"}}}}`,
		},
		{
			"group references undefined group",
			`{"type":"object","properties":{"f":{"type":"string","x-ui":{"group":"missing"}}}}`,
		},
		{
			"group references undefined group with other groups defined",
			`{"type":"object","x-ui":{"groups":[{"id":"billing","label":"Billing"}]},"properties":{"f":{"type":"string","x-ui":{"group":"shipping"}}}}`,
		},
		{
			"order references unknown property",
			`{"type":"object","x-ui":{"order":["name","nonexistent"]},"properties":{"name":{"type":"string"}}}`,
		},
		{
			"visible_when references unknown property",
			`{"type":"object","properties":{"f":{"type":"string","x-ui":{"visible_when":{"field":"ghost","equals":"val"}}}}}`,
		},
		{
			"root x-ui groups with empty id",
			`{"type":"object","x-ui":{"groups":[{"id":"","label":"Empty"}]},"properties":{"f":{"type":"string"}}}`,
		},
		{
			"duplicate group id",
			`{"type":"object","x-ui":{"groups":[{"id":"billing","label":"Billing"},{"id":"billing","label":"Billing 2"}]},"properties":{"f":{"type":"string"}}}`,
		},
		{
			"duplicate order entry",
			`{"type":"object","x-ui":{"order":["name","name"]},"properties":{"name":{"type":"string"}}}`,
		},
		{
			"visible_when missing field key",
			`{"type":"object","properties":{"f":{"type":"string","x-ui":{"visible_when":{"equals":"val"}}}}}`,
		},
		{
			"visible_when missing equals key",
			`{"type":"object","properties":{"a":{"type":"string"},"f":{"type":"string","x-ui":{"visible_when":{"field":"a"}}}}}`,
		},
		{
			"visible_when self-reference",
			`{"type":"object","properties":{"f":{"type":"string","x-ui":{"visible_when":{"field":"f","equals":"val"}}}}}`,
		},
		{
			"visible_when mutual dependency",
			`{"type":"object","properties":{"a":{"type":"string","x-ui":{"visible_when":{"field":"b","equals":"x"}}},"b":{"type":"string","x-ui":{"visible_when":{"field":"a","equals":"y"}}}}}`,
		},
		{
			"help_url with javascript scheme",
			`{"type":"object","properties":{"f":{"type":"string","x-ui":{"help_url":"javascript:alert(1)"}}}}`,
		},
		{
			"help_url with no host",
			`{"type":"object","properties":{"f":{"type":"string","x-ui":{"help_url":"https://"}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := base(tt.schema)
			_, err := ParseManifest([]byte(input))
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestParseManifest_XUIVisibleWhenEqualsNull(t *testing.T) {
	// visible_when with equals: null should be valid (means "show when field is unset").
	input := `{
		"id": "x",
		"name": "X",
		"actions": [{
			"action_type": "x.do",
			"name": "Do",
			"parameters_schema": {
				"type": "object",
				"properties": {
					"a": {"type": "string"},
					"b": {"type": "string", "x-ui": {"visible_when": {"field": "a", "equals": null}}}
				}
			}
		}]
	}`

	_, err := ParseManifest([]byte(input))
	if err != nil {
		t.Fatalf("visible_when with equals:null should be valid, got: %v", err)
	}
}

func TestParseManifest_XUIAllWidgetTypes(t *testing.T) {
	// All valid widget types should be accepted.
	for _, widget := range []string{"text", "select", "textarea", "toggle", "number", "date"} {
		t.Run(widget, func(t *testing.T) {
			input := fmt.Sprintf(`{
				"id": "x",
				"name": "X",
				"actions": [{
					"action_type": "x.do",
					"name": "Do",
					"parameters_schema": {
						"type": "object",
						"properties": {
							"f": {"type": "string", "x-ui": {"widget": %q}}
						}
					}
				}]
			}`, widget)
			_, err := ParseManifest([]byte(input))
			if err != nil {
				t.Fatalf("widget %q should be valid, got: %v", widget, err)
			}
		})
	}
}
