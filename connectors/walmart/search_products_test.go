package walmart

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchProducts_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/search" {
			t.Errorf("path = %s, want /search", r.URL.Path)
		}

		q := r.URL.Query()
		if got := q.Get("query"); got != "paper towels" {
			t.Errorf("query = %q, want %q", got, "paper towels")
		}
		if got := q.Get("numItems"); got != "5" {
			t.Errorf("numItems = %q, want %q", got, "5")
		}
		if got := q.Get("sort"); got != "price" {
			t.Errorf("sort = %q, want %q", got, "price")
		}
		if got := q.Get("order"); got != "asc" {
			t.Errorf("order = %q, want %q", got, "asc")
		}
		if got := q.Get("format"); got != "json" {
			t.Errorf("format = %q, want %q", got, "json")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"query":      "paper towels",
			"totalResults": 100,
			"items": []map[string]any{
				{
					"itemId":       12345,
					"name":         "Bounty Paper Towels",
					"salePrice":    12.99,
					"addToCartUrl": "https://www.walmart.com/cart/add?items=12345",
				},
			},
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.search_products"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.search_products",
		Parameters:  json.RawMessage(`{"query":"paper towels","limit":5,"sort":"price","order":"asc"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 item, got %v", data["items"])
	}
}

func TestSearchProducts_Defaults(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("numItems"); got != "10" {
			t.Errorf("numItems = %q, want %q (default)", got, "10")
		}
		if got := q.Get("sort"); got != "" {
			t.Errorf("sort = %q, want empty (not set)", got)
		}
		if got := q.Get("order"); got != "" {
			t.Errorf("order = %q, want empty (not set)", got)
		}
		if got := q.Get("start"); got != "" {
			t.Errorf("start = %q, want empty (not set)", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.search_products",
		Parameters:  json.RawMessage(`{"query":"laptop"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchProducts_WithCategory(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("categoryId"); got != "3944" {
			t.Errorf("categoryId = %q, want %q", got, "3944")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.search_products",
		Parameters:  json.RawMessage(`{"query":"milk","category_id":"3944"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchProducts_ValidationErrors(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["walmart.search_products"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing query", `{"limit":10}`},
		{"invalid sort", `{"query":"test","sort":"invalid"}`},
		{"invalid order", `{"query":"test","order":"invalid"}`},
		{"limit too high", `{"query":"test","limit":50}`},
		{"limit negative", `{"query":"test","limit":-1}`},
		{"negative start", `{"query":"test","start":-1}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "walmart.search_products",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestSearchProducts_APIError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(walmartErrorResponse(400, "Invalid query"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.search_products",
		Parameters:  json.RawMessage(`{"query":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestSearchProducts_RateLimit(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write(walmartErrorResponse(429, "rate limit exceeded"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.search_products",
		Parameters:  json.RawMessage(`{"query":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestSearchProducts_AuthError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(walmartErrorResponse(401, "Unauthorized"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.search_products"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.search_products",
		Parameters:  json.RawMessage(`{"query":"test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}
