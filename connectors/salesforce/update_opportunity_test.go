package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateOpportunity_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Opportunity/006xx0000001abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["StageName"] != "Closed Won" {
			t.Errorf("expected StageName 'Closed Won', got %v", body["StageName"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateOpportunityAction{conn: conn}

	amount := 75000.0
	params, _ := json.Marshal(updateOpportunityParams{
		RecordID:  "006xx0000001abc",
		StageName: "Closed Won",
		Amount:    &amount,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_opportunity",
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
	if data["record_id"] != "006xx0000001abc" {
		t.Errorf("expected record_id '006xx0000001abc', got %v", data["record_id"])
	}
	if data["success"] != true {
		t.Errorf("expected success true, got %v", data["success"])
	}
}

func TestUpdateOpportunity_MissingRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateOpportunityAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"stage_name": "Closed Won"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_opportunity",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing record_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateOpportunity_NoFieldsToUpdate(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateOpportunityAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"record_id": "006xx0000001abc"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_opportunity",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when no fields provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateOpportunity_InvalidRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateOpportunityAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"record_id":  "not-valid",
		"stage_name": "Closed Won",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_opportunity",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid record_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
