package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestMoveCard_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/cards/card123" {
			t.Errorf("expected path /cards/card123, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["idList"] != "done-list" {
			t.Errorf("expected idList=done-list, got %v", reqBody["idList"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":       "card123",
			"name":     "Card",
			"idList":   "done-list",
			"shortUrl": "https://trello.com/c/abc123",
			"url":      "https://trello.com/c/abc123/1-card",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.move_card"]

	params, _ := json.Marshal(moveCardParams{
		CardID: "card123",
		ListID: "done-list",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.move_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["idList"] != "done-list" {
		t.Errorf("expected idList=done-list, got %v", data["idList"])
	}
}

func TestMoveCard_WithPosition(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["pos"] != "top" {
			t.Errorf("expected pos=top, got %v", reqBody["pos"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "card123", "name": "Card", "idList": "list456", "shortUrl": "https://trello.com/c/abc", "url": "https://trello.com/c/abc/1"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.move_card"]

	params, _ := json.Marshal(moveCardParams{
		CardID: "card123",
		ListID: "list456",
		Pos:    "top",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.move_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMoveCard_MissingCardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.move_card"]

	params, _ := json.Marshal(map[string]string{"list_id": "list123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.move_card",
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

func TestMoveCard_MissingListID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.move_card"]

	params, _ := json.Marshal(map[string]string{"card_id": "card123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.move_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing list_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
