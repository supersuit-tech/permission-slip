package calendly

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCalendlyConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "calendly" {
		t.Errorf("expected ID 'calendly', got %q", c.ID())
	}
}

func TestCalendlyConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"calendly.list_event_types",
		"calendly.create_scheduling_link",
		"calendly.list_scheduled_events",
		"calendly.cancel_event",
		"calendly.get_event",
		"calendly.list_available_times",
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

func TestCalendlyConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(usersmeResponse{
			Resource: struct {
				URI  string `json:"uri"`
				Name string `json:"name"`
			}{URI: "https://api.calendly.com/users/abc123", Name: "Test User"},
		})
	}))
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)

	err := c.ValidateCredentials(t.Context(), validCreds())
	if err != nil {
		t.Errorf("ValidateCredentials() returned error: %v", err)
	}
}

func TestCalendlyConnector_ValidateCredentials_MissingKey(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
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

func TestCalendlyConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*CalendlyConnector)(nil)
	var _ connectors.ManifestProvider = (*CalendlyConnector)(nil)
}

func TestCalendlyConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "calendly" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "calendly")
	}
	if m.Name != "Calendly" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Calendly")
	}
	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"calendly.list_event_types",
		"calendly.create_scheduling_link",
		"calendly.list_scheduled_events",
		"calendly.cancel_event",
		"calendly.get_event",
		"calendly.list_available_times",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "calendly" {
		t.Errorf("credential service = %q, want %q", cred.Service, "calendly")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestCalendlyConnector_ActionsMatchManifest(t *testing.T) {
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

func TestDoJSON_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-calendly-api-key-123" {
			t.Errorf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/users/me" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"resource":{"uri":"https://api.calendly.com/users/abc123","name":"Test User"}}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp usersmeResponse

	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if err != nil {
		t.Fatalf("doJSON() returned error: %v", err)
	}
	if resp.Resource.URI != "https://api.calendly.com/users/abc123" {
		t.Errorf("resp.Resource.URI = %q, want %q", resp.Resource.URI, "https://api.calendly.com/users/abc123")
	}
}

func TestDoJSON_PostWithBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %q", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Content-Type: %q", r.Header.Get("Content-Type"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding request body: %v", err)
			return
		}
		if body["owner_type"] != "EventType" {
			t.Errorf("body owner_type = %q, want %q", body["owner_type"], "EventType")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"resource":{"booking_url":"https://calendly.com/d/abc","owner":"evt","owner_type":"EventType"}}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	reqBody := map[string]any{"owner_type": "EventType", "max_event_count": 1}
	var resp calendlySchedulingLinkResponse

	err := conn.doJSON(t.Context(), validCreds(), http.MethodPost, srv.URL+"/scheduling_links", reqBody, &resp)
	if err != nil {
		t.Fatalf("doJSON() returned error: %v", err)
	}
	if resp.Resource.BookingURL != "https://calendly.com/d/abc" {
		t.Errorf("resp.Resource.BookingURL = %q, want %q", resp.Resource.BookingURL, "https://calendly.com/d/abc")
	}
}

func TestDoJSON_MissingToken(t *testing.T) {
	t.Parallel()
	conn := newForTest(http.DefaultClient, "http://localhost")
	err := conn.doJSON(t.Context(), connectors.NewCredentials(map[string]string{}), http.MethodGet, "http://localhost/test", nil, nil)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_AuthError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"title":"Unauthenticated","message":"Invalid token."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"title":"Forbidden","message":"Insufficient permissions."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for 403, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"title":"Rate Limit Exceeded","message":"Too many requests."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/event_types", nil, &resp)
	if !connectors.IsRateLimitError(err) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want 30s", rle.RetryAfter)
		}
	}
}

func TestCheckResponse_BadRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"title":"Invalid Argument","message":"Invalid parameter: start_time"}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/test", nil, &resp)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 400, got %T: %v", err, err)
	}
}

func TestCheckResponse_UnprocessableEntity(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"title":"Validation Error","message":"event_type is invalid"}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodPost, srv.URL+"/scheduling_links", map[string]any{}, &resp)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 422, got %T: %v", err, err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"title":"Resource Not Found","message":"The scheduled event does not exist."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/scheduled_events/999", nil, &resp)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"title":"Internal Server Error","message":"Something went wrong"}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError for 500, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimitNoRetryAfter(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"title":"Rate Limit Exceeded","message":"Too many requests."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if !connectors.IsRateLimitError(err) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != defaultRetryAfter {
			t.Errorf("RetryAfter = %v, want default %v", rle.RetryAfter, defaultRetryAfter)
		}
	}
}

func TestCheckResponse_MalformedErrorBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `not json`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/test", nil, &resp)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 400 with malformed body, got %T: %v", err, err)
	}
}
