package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateCard_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/cards" {
			t.Errorf("expected path /cards, got %s", r.URL.Path)
		}
		// Verify auth query params.
		if r.URL.Query().Get("key") != "test-api-key-123" {
			t.Errorf("expected key=test-api-key-123, got %q", r.URL.Query().Get("key"))
		}
		if r.URL.Query().Get("token") != "test-token-456" {
			t.Errorf("expected token=test-token-456, got %q", r.URL.Query().Get("token"))
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected JSON content type, got %q", got)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if reqBody["idList"] != testListID {
			t.Errorf("expected idList=%s, got %v", testListID, reqBody["idList"])
		}
		if reqBody["name"] != "Test Card" {
			t.Errorf("expected name=Test Card, got %v", reqBody["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":       testCardID,
			"name":     "Test Card",
			"shortUrl": "https://trello.com/c/abc123",
			"url":      "https://trello.com/c/abc123/1-test-card",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.create_card"]

	params, _ := json.Marshal(createCardParams{
		ListID: testListID,
		Name:   "Test Card",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != testCardID {
		t.Errorf("expected id=%s, got %v", testCardID, data["id"])
	}
	if data["shortUrl"] != "https://trello.com/c/abc123" {
		t.Errorf("expected shortUrl, got %v", data["shortUrl"])
	}
}

func TestCreateCard_WithOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		if reqBody["desc"] != "A description" {
			t.Errorf("expected desc, got %v", reqBody["desc"])
		}
		if reqBody["pos"] != "top" {
			t.Errorf("expected pos=top, got %v", reqBody["pos"])
		}
		if reqBody["due"] != "2026-12-31T00:00:00.000Z" {
			t.Errorf("expected due date, got %v", reqBody["due"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": testCardID, "name": "Full Card", "shortUrl": "https://trello.com/c/xyz", "url": "https://trello.com/c/xyz/1"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.create_card"]

	params, _ := json.Marshal(createCardParams{
		ListID: testListID,
		Name:   "Full Card",
		Desc:   "A description",
		Pos:    "top",
		Due:    "2026-12-31T00:00:00.000Z",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCard_MissingListID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.create_card"]

	params, _ := json.Marshal(map[string]string{"name": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_card",
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

func TestCreateCard_InvalidListID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.create_card"]

	params, _ := json.Marshal(map[string]string{"list_id": "not-a-valid-id", "name": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid list_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCard_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.create_card"]

	params, _ := json.Marshal(map[string]string{"list_id": testListID})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_card",
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

func TestCreateCard_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.create_card"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_card",
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

func TestCreateCard_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Rate limit exceeded"))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.create_card"]

	params, _ := json.Marshal(createCardParams{ListID: testListID, Name: "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
	var rlErr *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rlErr) {
		if rlErr.RetryAfter.Seconds() != 10 {
			t.Errorf("expected RetryAfter 10s, got %v", rlErr.RetryAfter)
		}
	}
}

func TestCreateCard_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized permission requested"))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.create_card"]

	params, _ := json.Marshal(createCardParams{ListID: testListID, Name: "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.create_card",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}
