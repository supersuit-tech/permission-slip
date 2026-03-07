package walmart

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetProduct_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/items/12345" {
			t.Errorf("path = %s, want /items/12345", r.URL.Path)
		}
		if got := r.URL.Query().Get("format"); got != "json" {
			t.Errorf("format = %q, want %q", got, "json")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"itemId":          12345,
			"name":            "Bounty Paper Towels",
			"salePrice":       12.99,
			"customerRating":  "4.5",
			"numReviews":      1234,
			"availableOnline": true,
			"addToCartUrl":    "https://www.walmart.com/cart/add?items=12345",
			"thumbnailImage":  "https://i5.walmartimages.com/test.jpg",
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.get_product"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.get_product",
		Parameters:  json.RawMessage(`{"item_id":"12345"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if _, ok := data["addToCartUrl"]; !ok {
		t.Error("expected addToCartUrl in response")
	}
}

func TestGetProduct_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["walmart.get_product"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.get_product",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write(walmartErrorResponse(404, "Item not found"))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.get_product"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.get_product",
		Parameters:  json.RawMessage(`{"item_id":"99999"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}
