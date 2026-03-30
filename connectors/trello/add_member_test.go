package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const testMemberID = "507f1f77bcf86cd799439077"

func TestAddMember_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/cards/"+testCardID+"/idMembers" {
			t.Errorf("path = %s, want /cards/%s/idMembers", r.URL.Path, testCardID)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["value"] != testMemberID {
			t.Errorf("value = %v, want %s", reqBody["value"], testMemberID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{testMemberID})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.add_member"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.add_member",
		Parameters:  json.RawMessage(`{"card_id":"` + testCardID + `","member_id":"` + testMemberID + `"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "added" {
		t.Errorf("status = %v, want added", data["status"])
	}
	members, ok := data["id_members"].([]any)
	if !ok || len(members) != 1 {
		t.Errorf("id_members = %v, want 1 item", data["id_members"])
	}
}

func TestAddMember_InvalidMemberID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.add_member"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.add_member",
		Parameters:  json.RawMessage(`{"card_id":"` + testCardID + `","member_id":"bad"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
