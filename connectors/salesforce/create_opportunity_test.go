package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateOpportunity_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Opportunity/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["Name"] != "Big Deal" {
			t.Errorf("expected Name 'Big Deal', got %v", body["Name"])
		}
		if body["StageName"] != "Prospecting" {
			t.Errorf("expected StageName 'Prospecting', got %v", body["StageName"])
		}
		if body["CloseDate"] != "2026-12-31" {
			t.Errorf("expected CloseDate '2026-12-31', got %v", body["CloseDate"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sfCreateResponse{ID: "006xx0000001abc", Success: true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createOpportunityAction{conn: conn}

	amount := 50000.0
	params, _ := json.Marshal(createOpportunityParams{
		Name:      "Big Deal",
		StageName: "Prospecting",
		CloseDate: "2026-12-31",
		Amount:    &amount,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_opportunity",
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
	if data["id"] != "006xx0000001abc" {
		t.Errorf("expected id '006xx0000001abc', got %v", data["id"])
	}
	if data["success"] != true {
		t.Errorf("expected success true, got %v", data["success"])
	}
	if data["record_url"] != "https://myorg.salesforce.com/006xx0000001abc" {
		t.Errorf("expected record_url, got %v", data["record_url"])
	}
}

func TestCreateOpportunity_MissingRequiredFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createOpportunityAction{conn: conn}

	tests := []struct {
		name   string
		params map[string]any
	}{
		{"missing name", map[string]any{"stage_name": "Prospecting", "close_date": "2026-12-31"}},
		{"missing stage_name", map[string]any{"name": "Deal", "close_date": "2026-12-31"}},
		{"missing close_date", map[string]any{"name": "Deal", "stage_name": "Prospecting"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(tt.params)
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "salesforce.create_opportunity",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error for missing required field")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestCreateOpportunity_InvalidAccountID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createOpportunityAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"name":       "Deal",
		"stage_name": "Prospecting",
		"close_date": "2026-12-31",
		"account_id": "not-a-valid-id",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_opportunity",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid account_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateOpportunity_InvalidCloseDate(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createOpportunityAction{conn: conn}

	tests := []struct {
		name      string
		closeDate string
	}{
		{"wrong format", "31/12/2026"},
		{"invalid date", "2026-02-30"},
		{"just year", "2026"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(map[string]any{
				"name":       "Deal",
				"stage_name": "Prospecting",
				"close_date": tt.closeDate,
			})
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "salesforce.create_opportunity",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatalf("expected error for invalid close_date %q", tt.closeDate)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestCreateOpportunity_NegativeAmount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createOpportunityAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"name":       "Deal",
		"stage_name": "Prospecting",
		"close_date": "2026-12-31",
		"amount":     -100.0,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_opportunity",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

// TestCreateOpportunity_ExplicitZeroAmount verifies that amount=0 is sent to
// Salesforce rather than omitted. This is only possible with *float64; a plain
// float64 with omitempty would silently drop a legitimate zero-amount.
func TestCreateOpportunity_ExplicitZeroAmount(t *testing.T) {
	t.Parallel()

	amountSent := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if _, ok := body["Amount"]; ok {
			amountSent = true
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sfCreateResponse{ID: "006xx0000001abc", Success: true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createOpportunityAction{conn: conn}

	zero := 0.0
	params, _ := json.Marshal(createOpportunityParams{
		Name:      "Zero Budget Deal",
		StageName: "Prospecting",
		CloseDate: "2026-12-31",
		Amount:    &zero,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_opportunity",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !amountSent {
		t.Error("expected Amount=0 to be sent to Salesforce, but it was omitted")
	}
}
