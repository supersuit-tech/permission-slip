package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestFollowUser_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/users/99/following" {
			t.Errorf("path = %s, want /users/99/following", r.URL.Path)
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["target_user_id"] != "42" {
			t.Errorf("target_user_id = %s, want 42", body["target_user_id"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"following":     true,
				"pending_follow": false,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.follow_user"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.follow_user",
		Parameters:  json.RawMessage(`{"user_id":"99","target_user_id":"42"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["following"] != true {
		t.Errorf("following = %v, want true", data["following"])
	}
}

func TestFollowUser_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.follow_user"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.follow_user",
		Parameters:  json.RawMessage(`{"user_id":"99"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUnfollowUser_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/users/99/following/42" {
			t.Errorf("path = %s, want /users/99/following/42", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"following": false},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.unfollow_user"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.unfollow_user",
		Parameters:  json.RawMessage(`{"user_id":"99","target_user_id":"42"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["following"] != false {
		t.Errorf("following = %v, want false", data["following"])
	}
}
