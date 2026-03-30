package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateChecklist_Success(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")

		n := callCount.Add(1)
		switch {
		case n == 1 && r.URL.Path == "/checklists":
			// Create checklist request.
			body, _ := io.ReadAll(r.Body)
			var reqBody map[string]string
			json.Unmarshal(body, &reqBody)
			if reqBody["idCard"] != testCardID {
				t.Errorf("expected idCard=%s, got %q", testCardID, reqBody["idCard"])
			}
			if reqBody["name"] != "TODO" {
				t.Errorf("expected name=TODO, got %q", reqBody["name"])
			}
			json.NewEncoder(w).Encode(map[string]any{
				"id":   testChecklistID,
				"name": "TODO",
			})

		case n == 2 && r.URL.Path == "/checklists/"+testChecklistID+"/checkItems":
			body, _ := io.ReadAll(r.Body)
			var reqBody map[string]string
			json.Unmarshal(body, &reqBody)
			json.NewEncoder(w).Encode(map[string]any{
				"id":   "item1",
				"name": reqBody["name"],
			})

		case n == 3 && r.URL.Path == "/checklists/"+testChecklistID+"/checkItems":
			body, _ := io.ReadAll(r.Body)
			var reqBody map[string]string
			json.Unmarshal(body, &reqBody)
			json.NewEncoder(w).Encode(map[string]any{
				"id":   "item2",
				"name": reqBody["name"],
			})

		default:
			t.Errorf("unexpected request #%d: %s %s", n, r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.create_checklist"]

	params, _ := json.Marshal(createChecklistParams{
		CardID: testCardID,
		Name:   "TODO",
		Items:  []string{"Buy milk", "Walk dog"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_checklist",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["id"] != testChecklistID {
		t.Errorf("expected id=%s, got %v", testChecklistID, data["id"])
	}
	items, ok := data["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got %T", data["items"])
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestCreateChecklist_NoItems(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checklists" {
			t.Errorf("expected path /checklists, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   testChecklistID,
			"name": "Empty List",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.create_checklist"]

	params, _ := json.Marshal(createChecklistParams{
		CardID: testCardID,
		Name:   "Empty List",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_checklist",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["id"] != testChecklistID {
		t.Errorf("expected id=%s, got %v", testChecklistID, data["id"])
	}
}

func TestCreateChecklist_MissingCardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.create_checklist"]

	params, _ := json.Marshal(map[string]string{"name": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_checklist",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing card_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateChecklist_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.create_checklist"]

	params, _ := json.Marshal(map[string]string{"card_id": testCardID})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_checklist",
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
