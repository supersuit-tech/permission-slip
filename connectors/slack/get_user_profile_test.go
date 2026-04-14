package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetUserProfile_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users.info" {
			t.Errorf("path = %s, want /users.info", r.URL.Path)
		}
		if r.URL.Query().Get("user") != "U01234567" {
			t.Errorf("user = %q, want U01234567", r.URL.Query().Get("user"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"user": map[string]any{
				"id": "U01234567", "name": "octocat", "real_name": "The Octocat",
				"profile": map[string]any{
					"display_name": "octo", "real_name": "The Octocat", "email": "o@example.com",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getUserProfileAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.get_user_profile",
		Parameters:  json.RawMessage(`{"user_id":"U01234567"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "U01234567" {
		t.Errorf("id = %v, want U01234567", data["id"])
	}
}

func TestGetUserProfile_MissingOrInvalidUserID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getUserProfileAction{conn: conn}

	tests := []struct {
		name   string
		params string
	}{
		{"missing user_id", `{}`},
		{"username instead of ID", `{"user_id":"octocat"}`},
		{"email instead of ID", `{"user_id":"o@example.com"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "slack.get_user_profile",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

func TestGetUserProfile_MissingUserInResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getUserProfileAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.get_user_profile",
		Parameters:  json.RawMessage(`{"user_id":"U01234567"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when user field absent, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}
