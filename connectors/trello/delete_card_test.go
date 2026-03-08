package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteCard_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/cards/"+testCardID {
			t.Errorf("path = %s, want /cards/%s", r.URL.Path, testCardID)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.delete_card"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.delete_card",
		Parameters:  json.RawMessage(`{"card_id":"` + testCardID + `"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["status"] != "deleted" {
		t.Errorf("status = %v, want deleted", data["status"])
	}
}

func TestDeleteCard_InvalidCardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.delete_card"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.delete_card",
		Parameters:  json.RawMessage(`{"card_id":"bad-id"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
