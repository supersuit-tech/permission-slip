package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestConvertLead_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Lead/00Qxx0000001abc/convert" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body sfConvertLeadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.LeadID != "00Qxx0000001abc" {
			t.Errorf("expected leadId '00Qxx0000001abc', got %v", body.LeadID)
		}
		if body.ConvertedStatus != "Closed - Converted" {
			t.Errorf("expected convertedStatus 'Closed - Converted', got %v", body.ConvertedStatus)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sfConvertLeadResponse{
			LeadID:        "00Qxx0000001abc",
			AccountID:     "001xx0000001xyz",
			ContactID:     "003xx0000001xyz",
			OpportunityID: "006xx0000001xyz",
			Success:       true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &convertLeadAction{conn: conn}

	params, _ := json.Marshal(convertLeadParams{
		LeadID:          "00Qxx0000001abc",
		ConvertedStatus: "Closed - Converted",
		OpportunityName: "New Opportunity",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.convert_lead",
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
	if data["lead_id"] != "00Qxx0000001abc" {
		t.Errorf("expected lead_id '00Qxx0000001abc', got %v", data["lead_id"])
	}
	if data["account_id"] != "001xx0000001xyz" {
		t.Errorf("expected account_id, got %v", data["account_id"])
	}
	if data["opportunity_id"] != "006xx0000001xyz" {
		t.Errorf("expected opportunity_id, got %v", data["opportunity_id"])
	}
	if data["success"] != true {
		t.Errorf("expected success true, got %v", data["success"])
	}
	// Record URLs should be included for non-empty IDs.
	if data["account_url"] != "https://myorg.salesforce.com/001xx0000001xyz" {
		t.Errorf("expected account_url, got %v", data["account_url"])
	}
	if data["contact_url"] != "https://myorg.salesforce.com/003xx0000001xyz" {
		t.Errorf("expected contact_url, got %v", data["contact_url"])
	}
	if data["opportunity_url"] != "https://myorg.salesforce.com/006xx0000001xyz" {
		t.Errorf("expected opportunity_url, got %v", data["opportunity_url"])
	}
}

func TestConvertLead_MissingLeadID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &convertLeadAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"converted_status": "Closed - Converted"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.convert_lead",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing lead_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestConvertLead_MissingConvertedStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &convertLeadAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"lead_id": "00Qxx0000001abc"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.convert_lead",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing converted_status")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestConvertLead_InvalidLeadID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &convertLeadAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"lead_id":          "bad-id",
		"converted_status": "Closed - Converted",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.convert_lead",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid lead_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
