package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateCollection_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/custom_collections.json" {
			t.Errorf("path = %s, want /custom_collections.json", got)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		collection := reqBody["custom_collection"]
		if collection["title"] != "Summer Sale" {
			t.Errorf("title = %v, want %q", collection["title"], "Summer Sale")
		}
		if collection["sort_order"] != "best-selling" {
			t.Errorf("sort_order = %v, want %q", collection["sort_order"], "best-selling")
		}
		if collection["published"] != true {
			t.Errorf("published = %v, want true", collection["published"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"custom_collection": map[string]any{
				"id": 6001, "title": "Summer Sale", "sort_order": "best-selling",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_collection"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_collection",
		Parameters:  json.RawMessage(`{"title":"Summer Sale","sort_order":"best-selling","published":true}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["custom_collection"]; !ok {
		t.Error("result missing 'custom_collection' key")
	}
}

func TestCreateCollection_TitleOnly(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		collection := reqBody["custom_collection"]
		if _, ok := collection["sort_order"]; ok {
			t.Error("sort_order should not be present when not provided")
		}
		if _, ok := collection["published"]; ok {
			t.Error("published should not be present when not provided")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"custom_collection": map[string]any{"id": 6002, "title": "Basics"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_collection"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_collection",
		Parameters:  json.RawMessage(`{"title":"Basics"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestCreateCollection_WithImage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		collection := reqBody["custom_collection"]
		image, ok := collection["image"].(map[string]interface{})
		if !ok {
			t.Fatal("image not present or wrong type")
		}
		if image["src"] != "https://example.com/banner.jpg" {
			t.Errorf("image src = %v, want https://example.com/banner.jpg", image["src"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"custom_collection": map[string]any{"id": 6003},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_collection"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_collection",
		Parameters:  json.RawMessage(`{"title":"With Image","image":{"src":"https://example.com/banner.jpg","alt":"Banner"}}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestCreateCollection_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_collection"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_collection",
		Parameters:  json.RawMessage(`{"sort_order":"manual"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCollection_InvalidSortOrder(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_collection"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_collection",
		Parameters:  json.RawMessage(`{"title":"Test","sort_order":"random"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCollection_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_collection"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_collection",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCollection_APIValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"title":["has already been taken"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_collection"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_collection",
		Parameters:  json.RawMessage(`{"title":"Duplicate"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 422, got %T: %v", err, err)
	}
}
