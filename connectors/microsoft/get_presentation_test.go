package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetPresentation_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/me/drive/items/item-abc-123" {
			t.Errorf("expected path /me/drive/items/item-abc-123, got %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "item-abc-123",
			"name":   "Q4 Review.pptx",
			"webUrl": "https://onedrive.live.com/edit.aspx?id=item-abc-123",
			"size":   2048576,
			"lastModifiedBy": map[string]any{
				"user": map[string]string{
					"displayName": "Jane Smith",
				},
			},
			"lastModifiedDateTime": "2024-03-15T14:30:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(getPresentationParams{
		ItemID: "item-abc-123",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_presentation",
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
	if data["item_id"] != "item-abc-123" {
		t.Errorf("expected item_id 'item-abc-123', got %v", data["item_id"])
	}
	if data["name"] != "Q4 Review.pptx" {
		t.Errorf("expected name 'Q4 Review.pptx', got %v", data["name"])
	}
	if data["web_url"] != "https://onedrive.live.com/edit.aspx?id=item-abc-123" {
		t.Errorf("expected web_url, got %v", data["web_url"])
	}
	if data["last_modified_by"] != "Jane Smith" {
		t.Errorf("expected last_modified_by 'Jane Smith', got %v", data["last_modified_by"])
	}
	if data["last_modified"] != "2024-03-15T14:30:00Z" {
		t.Errorf("expected last_modified '2024-03-15T14:30:00Z', got %v", data["last_modified"])
	}
	// JSON numbers decode as float64.
	if size, ok := data["size"].(float64); !ok || size != 2048576 {
		t.Errorf("expected size 2048576, got %v", data["size"])
	}
}

func TestGetPresentation_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(getPresentationParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetPresentation_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "itemNotFound",
				"message": "Item does not exist",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(getPresentationParams{
		ItemID: "nonexistent-item",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found item")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestGetPresentation_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token has expired",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getPresentationAction{conn: conn}

	params, _ := json.Marshal(getPresentationParams{
		ItemID: "item-123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_presentation",
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

func TestGetPresentation_ItemIDPathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPresentationAction{conn: conn}

	// A crafted item_id with path traversal could hit a different Graph endpoint.
	params, _ := json.Marshal(getPresentationParams{
		ItemID: "foo/../me/messages",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for item_id with path traversal")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetPresentation_ItemIDQueryInjection(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPresentationAction{conn: conn}

	// A crafted item_id with ? could alter the query string.
	params, _ := json.Marshal(getPresentationParams{
		ItemID: "item123?$select=content",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for item_id with query injection")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetPresentation_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPresentationAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_presentation",
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
