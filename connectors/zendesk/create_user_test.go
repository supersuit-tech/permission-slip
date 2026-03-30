package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateUser_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/users.json" {
			t.Errorf("expected path /users.json, got %s", r.URL.Path)
		}

		var body map[string]zendeskUser
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		u := body["user"]
		if u.Name != "Jane Doe" {
			t.Errorf("expected name Jane Doe, got %q", u.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userResponse{
			User: zendeskUser{ID: 9001, Name: "Jane Doe", Email: "jane@example.com", Role: "end-user"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createUserAction{conn: conn}

	params, _ := json.Marshal(createUserParams{
		Name:  "Jane Doe",
		Email: "jane@example.com",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.create_user",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data userResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.User.ID != 9001 {
		t.Errorf("expected id 9001, got %d", data.User.ID)
	}
}

func TestCreateUser_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createUserAction{conn: conn}

	params, _ := json.Marshal(createUserParams{Email: "jane@example.com"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.create_user",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateUser_InvalidRole(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createUserAction{conn: conn}

	params, _ := json.Marshal(createUserParams{Name: "Jane Doe", Role: "superuser"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.create_user",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
