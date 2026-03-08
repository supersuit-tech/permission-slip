package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetUser_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/9001.json" {
			t.Errorf("expected path /users/9001.json, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userResponse{
			User: zendeskUser{ID: 9001, Name: "Jane Doe", Email: "jane@example.com"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getUserAction{conn: conn}

	params, _ := json.Marshal(getUserParams{UserID: 9001})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.get_user",
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
	if data.User.Name != "Jane Doe" {
		t.Errorf("expected name Jane Doe, got %q", data.User.Name)
	}
}

func TestGetUser_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getUserAction{conn: conn}

	params, _ := json.Marshal(getUserParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.get_user",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
