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
		expectedPath := "/cards/" + testCardID
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		if reqBody["idList"] != testListID {
			t.Errorf("expected idList=%s, got %v", testListID, reqBody["idList"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":       testCardID,
			"name":     "Card",
			"idList":   testListID,
			"shortUrl": "https://trello.com/c/abc123",
			"url":      "https://trello.com/c/abc123/1-card",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.move_card"]

	params, _ := json.Marshal(moveCardParams{
		CardID: testCardID,
		ListID: testListID,
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
	if data["idList"] != testListID {
		t.Errorf("expected idList=%s, got %v", testListID, data["idList"])
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
		json.NewEncoder(w).Encode(map[string]any{"id": testCardID, "name": "Card", "idList": testListID, "shortUrl": "https://trello.com/c/abc", "url": "https://trello.com/c/abc/1"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.move_card"]

	params, _ := json.Marshal(moveCardParams{
		CardID: testCardID,
		ListID: testListID,
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

	params, _ := json.Marshal(map[string]string{"list_id": testListID})

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

	params, _ := json.Marshal(map[string]string{"card_id": testCardID})

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

func TestMoveCard_InvalidCardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.move_card"]

	params, _ := json.Marshal(map[string]string{"card_id": "bad-id", "list_id": testListID})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.move_card",
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
