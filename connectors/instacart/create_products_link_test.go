package instacart

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateProductsLink_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/idp/v1/products/products_link" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key-instacart" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if _, ok := reqBody["line_items"]; !ok {
			t.Fatal("missing line_items in request")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"products_link_url":"https://customers.dev.instacart.tools/store/share/abc"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["instacart.create_products_link"]

	params := map[string]any{
		"title": "Weekly shop",
		"line_items": []map[string]any{
			{"name": "organic milk", "line_item_measurements": []map[string]any{{"quantity": 1, "unit": "each"}}},
		},
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  paramsJSON,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var out struct {
		ProductsLinkURL string `json:"products_link_url"`
	}
	if err := json.Unmarshal(result.Data, &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if out.ProductsLinkURL == "" {
		t.Fatal("empty products_link_url")
	}
}

func TestCreateProductsLink_StringLineItems(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			LineItems []map[string]any `json:"line_items"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(reqBody.LineItems) != 2 {
			t.Fatalf("line_items len = %d", len(reqBody.LineItems))
		}
		if reqBody.LineItems[0]["name"] != "milk" || reqBody.LineItems[1]["name"] != "eggs" {
			t.Fatalf("unexpected payload: %#v", reqBody.LineItems)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"products_link_url":"https://example.test/x"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["instacart.create_products_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"line_items":["milk","eggs"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestCreateProductsLink_EmptyTitleWhitespace(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["instacart.create_products_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"title":"   ","line_items":[{"name":"x"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("want ValidationError, got %T", err)
	}
}

func TestCreateProductsLink_TitleTooLong(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["instacart.create_products_link"]
	long := strings.Repeat("t", maxTitleLen+1)

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"title":"` + long + `","line_items":[{"name":"x"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("want ValidationError, got %T", err)
	}
}

func TestCreateProductsLink_MissingLineItems(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["instacart.create_products_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"title":"x"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("want ValidationError, got %T: %v", err, err)
	}
}

func TestCreateProductsLink_LineItemNameTooLong(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["instacart.create_products_link"]
	longName := make([]byte, maxLineItemNameChars+1)
	for i := range longName {
		longName[i] = 'a'
	}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"line_items":[{"name":"` + string(longName) + `"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("want ValidationError, got %T", err)
	}
}

func TestCreateProductsLink_EmptyLineItemName(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["instacart.create_products_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"line_items":[{"name":""}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("want ValidationError, got %T", err)
	}
}

func TestCreateProductsLink_InvalidLinkType(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["instacart.create_products_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"link_type":"other","line_items":[{"name":"a"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("want ValidationError, got %T", err)
	}
}

func TestCreateProductsLink_ItemsAlias(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			LineItems []map[string]any `json:"line_items"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(reqBody.LineItems) != 1 || reqBody.LineItems[0]["name"] != "bread" {
			t.Fatalf("unexpected payload: %#v", reqBody.LineItems)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"products_link_url":"https://example.test/y"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["instacart.create_products_link"]

	// Simulate the alias+normalize pipeline the API layer runs before Execute.
	raw := json.RawMessage(`{"items":["bread"]}`)
	if aliaser, ok := action.(connectors.ParameterAliaser); ok {
		raw = connectors.NormalizeParameters(aliaser.ParameterAliases(), raw)
	}
	if normalizer, ok := action.(connectors.Normalizer); ok {
		raw = normalizer.Normalize(raw)
	}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  raw,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestCreateProductsLink_EmptyProductsLinkURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"products_link_url":""}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["instacart.create_products_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"line_items":[{"name":"milk"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for empty products_link_url")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("want ExternalError, got %T: %v", err, err)
	}
}

func TestCreateProductsLink_AuthError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid token","code":401}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["instacart.create_products_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "instacart.create_products_link",
		Parameters:  json.RawMessage(`{"line_items":[{"name":"milk"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("want AuthError, got %T: %v", err, err)
	}
}
