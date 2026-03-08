package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateAd_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/act_123456789/ads" {
			t.Errorf("expected path /act_123456789/ads, got %s", r.URL.Path)
		}

		var body createAdRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Name != "Summer Sale Ad" {
			t.Errorf("expected name 'Summer Sale Ad', got %q", body.Name)
		}
		if body.AdSetID != "adset-456" {
			t.Errorf("expected adset_id 'adset-456', got %q", body.AdSetID)
		}
		if body.Status != "PAUSED" {
			t.Errorf("expected default status 'PAUSED', got %q", body.Status)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createAdResponse{ID: "ad-999"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createAdAction{conn: conn}

	params, _ := json.Marshal(createAdParams{
		AdAccountID: "123456789",
		Name:        "Summer Sale Ad",
		AdSetID:     "adset-456",
		CreativeID:  "creative-789",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_ad",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "ad-999" {
		t.Errorf("expected id 'ad-999', got %q", data["id"])
	}
}

func TestCreateAd_MissingAdSetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createAdAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"ad_account_id": "123456789",
		"name":          "Test Ad",
		"creative_id":   "creative-789",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_ad",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestCreateAd_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createAdAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"ad_account_id": "123456789",
		"name":          "Test Ad",
		"adset_id":      "adset-456",
		"creative_id":   "creative-789",
		"status":        "DELETED",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_ad",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for invalid status, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
