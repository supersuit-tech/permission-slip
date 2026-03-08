package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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

		w.WriteHeader(http.StatusOK)
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
	json.Unmarshal(result.Data, &data)
	if data["status"] != "added" {
		t.Errorf("status = %v, want added", data["status"])
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
