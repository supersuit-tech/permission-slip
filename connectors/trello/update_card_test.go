package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateCard_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		expectedPath := "/cards/" + testCardID
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["name"] != "Updated Name" {
			t.Errorf("expected name=Updated Name, got %v", reqBody["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":       testCardID,
			"name":     "Updated Name",
			"shortUrl": "https://trello.com/c/abc123",
			"url":      "https://trello.com/c/abc123/1-updated-name",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.update_card"]

	params, _ := json.Marshal(map[string]any{
		"card_id": testCardID,
		"name":    "Updated Name",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.update_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["id"] != testCardID {
		t.Errorf("expected id=%s, got %v", testCardID, data["id"])
	}
}

func TestUpdateCard_BooleanFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		if reqBody["dueComplete"] != true {
			t.Errorf("expected dueComplete=true, got %v", reqBody["dueComplete"])
		}
		if reqBody["closed"] != true {
			t.Errorf("expected closed=true, got %v", reqBody["closed"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": testCardID, "name": "Card", "shortUrl": "https://trello.com/c/abc", "url": "https://trello.com/c/abc/1"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.update_card"]

	bTrue := true
	params, _ := json.Marshal(updateCardParams{
		CardID:      testCardID,
		DueComplete: &bTrue,
		Closed:      &bTrue,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.update_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateCard_MissingCardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.update_card"]

	params, _ := json.Marshal(map[string]string{"name": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.update_card",
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

func TestUpdateCard_InvalidCardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.update_card"]

	params, _ := json.Marshal(map[string]string{"card_id": "short", "name": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.update_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid card_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateCard_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.update_card"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.update_card",
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
