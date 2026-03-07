package asana

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAsanaConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "asana" {
		t.Errorf("ID() = %q, want %q", got, "asana")
	}
}

func TestAsanaConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"asana.create_task",
		"asana.update_task",
		"asana.add_comment",
		"asana.complete_task",
		"asana.create_subtask",
		"asana.search_tasks",
	}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestAsanaConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "0/abc123"}),
			wantErr: false,
		},
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

func TestAsanaConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "asana" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "asana")
	}
	if m.Name != "Asana" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Asana")
	}
	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestAsanaConnector_ActionsMatchManifest(t *testing.T) {
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

func TestAsanaConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*AsanaConnector)(nil)
	var _ connectors.ManifestProvider = (*AsanaConnector)(nil)
}

func TestAsanaConnector_DoUnwrapsEnvelope(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer 0/abc123test" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer 0/abc123test")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"gid":  "12345",
				"name": "Test",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)

	var resp struct {
		GID  string `json:"gid"`
		Name string `json:"name"`
	}
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", nil, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp.GID != "12345" {
		t.Errorf("GID = %q, want %q", resp.GID, "12345")
	}
	if resp.Name != "Test" {
		t.Errorf("Name = %q, want %q", resp.Name, "Test")
	}
}

func TestAsanaConnector_DoWrapsRequestBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var envelope map[string]any
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		// Verify the request body is wrapped in {"data": ...}
		data, ok := envelope["data"]
		if !ok {
			t.Fatal("request body missing 'data' wrapper")
		}
		dataMap, ok := data.(map[string]any)
		if !ok {
			t.Fatalf("data is %T, want map", data)
		}
		if dataMap["name"] != "test" {
			t.Errorf("data.name = %v, want %q", dataMap["name"], "test")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"gid": "1"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp struct {
		GID string `json:"gid"`
	}
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/test", map[string]any{"name": "test"}, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

func TestAsanaConnector_ErrorMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		headers    map[string]string
		checkErr   func(t *testing.T, err error)
	}{
		{
			name:       "401 returns AuthError",
			statusCode: 401,
			body:       `{"errors":[{"message":"Not Authorized"}]}`,
			checkErr: func(t *testing.T, err error) {
				if !connectors.IsAuthError(err) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				}
			},
		},
		{
			name:       "429 returns RateLimitError",
			statusCode: 429,
			body:       `{"errors":[{"message":"Rate limit"}]}`,
			headers:    map[string]string{"Retry-After": "60"},
			checkErr: func(t *testing.T, err error) {
				if !connectors.IsRateLimitError(err) {
					t.Errorf("expected RateLimitError, got %T: %v", err, err)
				}
			},
		},
		{
			name:       "400 returns ValidationError",
			statusCode: 400,
			body:       `{"errors":[{"message":"Invalid request"}]}`,
			checkErr: func(t *testing.T, err error) {
				if !connectors.IsValidationError(err) {
					t.Errorf("expected ValidationError, got %T: %v", err, err)
				}
			},
		},
		{
			name:       "500 returns ExternalError",
			statusCode: 500,
			body:       `{"errors":[{"message":"Server error"}]}`,
			checkErr: func(t *testing.T, err error) {
				if !connectors.IsExternalError(err) {
					t.Errorf("expected ExternalError, got %T: %v", err, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.headers {
					w.Header().Set(k, v)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			conn := newForTest(srv.Client(), srv.URL)
			var resp struct{}
			err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", nil, &resp)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			tt.checkErr(t, err)
		})
	}
}
