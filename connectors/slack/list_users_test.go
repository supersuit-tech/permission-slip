package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListUsers_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/users.list" {
			t.Errorf("expected path /users.list, got %s", r.URL.Path)
		}

		var body listUsersRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Limit != 100 {
			t.Errorf("expected limit 100, got %d", body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"members": []map[string]any{
				{
					"id":        "U001",
					"name":      "jdoe",
					"real_name": "Jane Doe",
					"deleted":   false,
					"is_bot":    false,
					"is_admin":  true,
					"profile": map[string]string{
						"display_name": "Jane",
						"email":        "jane@example.com",
					},
				},
				{
					"id":        "U002",
					"name":      "bot_user",
					"real_name": "Bot",
					"deleted":   false,
					"is_bot":    true,
					"is_admin":  false,
					"profile": map[string]string{
						"display_name": "Helper Bot",
						"email":        "",
					},
				},
			},
			"response_metadata": map[string]string{
				"next_cursor": "dXNlcjpVMDAy",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUsersAction{conn: conn}

	params, _ := json.Marshal(listUsersParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_users",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listUsersResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(data.Users))
	}
	if data.Users[0].ID != "U001" {
		t.Errorf("expected first user ID 'U001', got %q", data.Users[0].ID)
	}
	if data.Users[0].Name != "jdoe" {
		t.Errorf("expected first user name 'jdoe', got %q", data.Users[0].Name)
	}
	if data.Users[0].Email != "jane@example.com" {
		t.Errorf("expected email 'jane@example.com', got %q", data.Users[0].Email)
	}
	if data.Users[1].IsBot != true {
		t.Error("expected second user to be a bot")
	}
	if data.NextCursor != "dXNlcjpVMDAy" {
		t.Errorf("expected next_cursor 'dXNlcjpVMDAy', got %q", data.NextCursor)
	}
}

func TestListUsers_WithPagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body listUsersRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Cursor != "dXNlcjpVMDAy" {
			t.Errorf("expected cursor 'dXNlcjpVMDAy', got %q", body.Cursor)
		}
		if body.Limit != 50 {
			t.Errorf("expected limit 50, got %d", body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"members": []map[string]any{},
			"response_metadata": map[string]string{
				"next_cursor": "",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUsersAction{conn: conn}

	params, _ := json.Marshal(listUsersParams{
		Limit:  50,
		Cursor: "dXNlcjpVMDAy",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_users",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listUsersResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Users) != 0 {
		t.Errorf("expected 0 users, got %d", len(data.Users))
	}
}

func TestListUsers_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listUsersAction{conn: conn}

	params, _ := json.Marshal(listUsersParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_users",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestListUsers_LimitOutOfRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listUsersAction{conn: conn}

	params, _ := json.Marshal(listUsersParams{Limit: 2000})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_users",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for limit out of range")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListUsers_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listUsersAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_users",
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
