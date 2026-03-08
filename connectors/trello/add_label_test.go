package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const testLabelID = "507f1f77bcf86cd799439055"

func TestAddLabel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/cards/"+testCardID+"/idLabels" {
			t.Errorf("path = %s, want /cards/%s/idLabels", r.URL.Path, testCardID)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["value"] != testLabelID {
			t.Errorf("value = %v, want %s", reqBody["value"], testLabelID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]string{testLabelID, "507f1f77bcf86cd799439066"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.add_label"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.add_label",
		Parameters:  json.RawMessage(`{"card_id":"` + testCardID + `","label_id":"` + testLabelID + `"}`),
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
	labels, ok := data["id_labels"].([]any)
	if !ok || len(labels) != 2 {
		t.Errorf("id_labels = %v, want 2 items", data["id_labels"])
	}
}

func TestAddLabel_InvalidCardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.add_label"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.add_label",
		Parameters:  json.RawMessage(`{"card_id":"bad","label_id":"` + testLabelID + `"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
