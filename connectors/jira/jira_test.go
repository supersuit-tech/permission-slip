package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Compile-time interface checks.
var (
	_ connectors.Connector = (*JiraConnector)(nil)
)

func TestJiraConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "jira" {
		t.Errorf("ID() = %q, want %q", c.ID(), "jira")
	}
}

func TestJiraConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "missing site",
			creds:   connectors.NewCredentials(map[string]string{"email": "user@example.com", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "site",
		},
		{
			name:    "missing email",
			creds:   connectors.NewCredentials(map[string]string{"site": "mysite", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "email",
		},
		{
			name:    "missing api_token",
			creds:   connectors.NewCredentials(map[string]string{"site": "mysite", "email": "user@example.com"}),
			wantErr: true,
			errMsg:  "api_token",
		},
		{
			name:    "empty site",
			creds:   connectors.NewCredentials(map[string]string{"site": "", "email": "user@example.com", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "site",
		},
	}

	c := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(context.Background(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestJiraConnector_Do_BasicAuth(t *testing.T) {
	t.Parallel()

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:test-api-token-123"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Errorf("Authorization = %q, want %q", got, wantAuth)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want %q", got, "application/json")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", nil, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("response status = %q, want %q", resp["status"], "ok")
	}
}

func TestJiraConnector_Do_PostBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "12345"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	body := map[string]string{"summary": "Test issue"}
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/issue", body, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "12345" {
		t.Errorf("id = %q, want %q", resp["id"], "12345")
	}
}

func TestJiraConnector_Do_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(ctx, validCreds(), http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestJiraConnector_Do_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := newForTest(nil, "http://localhost")
	tests := []struct {
		name  string
		creds connectors.Credentials
	}{
		{
			name:  "missing email",
			creds: connectors.NewCredentials(map[string]string{"site": "s", "api_token": "t"}),
		},
		{
			name:  "missing api_token",
			creds: connectors.NewCredentials(map[string]string{"site": "s", "email": "e"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := conn.do(t.Context(), tt.creds, http.MethodGet, "/test", nil, nil)
			if err == nil {
				t.Fatal("do() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
