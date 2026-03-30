package airtable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListBases_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/meta/bases" {
			t.Errorf("expected path /meta/bases, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer patTEST1234567890.abcdef" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"bases": []map[string]any{
				{"id": "appABC123", "name": "My CRM", "permissionLevel": "create"},
				{"id": "appDEF456", "name": "Inventory", "permissionLevel": "read"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listBasesAction{conn: conn}

	params, _ := json.Marshal(listBasesParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_bases",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listBasesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Bases) != 2 {
		t.Fatalf("expected 2 bases, got %d", len(data.Bases))
	}
	if data.Bases[0].ID != "appABC123" {
		t.Errorf("expected first base ID 'appABC123', got %q", data.Bases[0].ID)
	}
	if data.Bases[0].Name != "My CRM" {
		t.Errorf("expected first base name 'My CRM', got %q", data.Bases[0].Name)
	}
}

func TestListBases_Pagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") != "itr123" {
			t.Errorf("expected offset 'itr123', got %q", r.URL.Query().Get("offset"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"bases":  []map[string]any{},
			"offset": "itr456",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listBasesAction{conn: conn}

	params, _ := json.Marshal(listBasesParams{Offset: "itr123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_bases",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listBasesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Offset != "itr456" {
		t.Errorf("expected offset 'itr456', got %q", data.Offset)
	}
}

func TestListBases_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"type":"AUTHENTICATION_REQUIRED","message":"invalid token"}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listBasesAction{conn: conn}

	params, _ := json.Marshal(listBasesParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_bases",
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

func TestListBases_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listBasesAction{conn: conn}

	params, _ := json.Marshal(listBasesParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_bases",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestListBases_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listBasesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.list_bases",
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
