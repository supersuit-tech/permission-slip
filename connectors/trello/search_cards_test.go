package trello

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchCards_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/search" {
			t.Errorf("expected path /search, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "bug fix" {
			t.Errorf("expected query='bug fix', got %q", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("modelTypes") != "cards" {
			t.Errorf("expected modelTypes=cards, got %q", r.URL.Query().Get("modelTypes"))
		}
		if r.URL.Query().Get("cards_limit") != "10" {
			t.Errorf("expected cards_limit=10, got %q", r.URL.Query().Get("cards_limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"cards": []map[string]any{
				{"id": "card1", "name": "Fix bug #1"},
				{"id": "card2", "name": "Fix bug #2"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.search_cards"]

	params, _ := json.Marshal(searchCardsParams{
		Query: "bug fix",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.search_cards",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["count"] != float64(2) {
		t.Errorf("expected count=2, got %v", data["count"])
	}
	cards, ok := data["cards"].([]any)
	if !ok {
		t.Fatalf("expected cards array, got %T", data["cards"])
	}
	if len(cards) != 2 {
		t.Errorf("expected 2 cards, got %d", len(cards))
	}
}

func TestSearchCards_WithBoardFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("idBoards") != "board123" {
			t.Errorf("expected idBoards=board123, got %q", r.URL.Query().Get("idBoards"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"cards": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.search_cards"]

	params, _ := json.Marshal(searchCardsParams{
		Query:   "test",
		BoardID: "board123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.search_cards",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchCards_CustomLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("cards_limit") != "50" {
			t.Errorf("expected cards_limit=50, got %q", r.URL.Query().Get("cards_limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"cards": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.search_cards"]

	params, _ := json.Marshal(searchCardsParams{
		Query: "test",
		Limit: 50,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.search_cards",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchCards_MissingQuery(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.search_cards"]

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.search_cards",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing query")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchCards_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.search_cards"]

	params, _ := json.Marshal(searchCardsParams{
		Query: "test",
		Limit: 2000,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.search_cards",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid limit")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchCards_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.search_cards"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.search_cards",
		Parameters:  []byte(`{bad`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
