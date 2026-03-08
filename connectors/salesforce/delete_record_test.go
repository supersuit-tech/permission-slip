package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteRecord_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Lead/00Qxx0000001abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteRecordAction{conn: conn}

	params, _ := json.Marshal(deleteRecordParams{
		SObjectType: "Lead",
		RecordID:    "00Qxx0000001abc",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.delete_record",
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
	if data["record_id"] != "00Qxx0000001abc" {
		t.Errorf("expected record_id, got %v", data["record_id"])
	}
	if data["sobject_type"] != "Lead" {
		t.Errorf("expected sobject_type 'Lead', got %v", data["sobject_type"])
	}
	if data["success"] != true {
		t.Errorf("expected success true, got %v", data["success"])
	}
}

func TestDeleteRecord_MissingSObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"record_id": "00Qxx0000001abc"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.delete_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing sobject_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDeleteRecord_MissingRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"sobject_type": "Lead"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.delete_record",
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

func TestDeleteRecord_InvalidRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"sobject_type": "Lead",
		"record_id":    "not-valid",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.delete_record",
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

func TestDeleteRecord_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode([]sfAPIError{{ErrorCode: "NOT_FOUND", Message: "Record not found"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteRecordAction{conn: conn}

	params, _ := json.Marshal(deleteRecordParams{
		SObjectType: "Lead",
		RecordID:    "00Qxx0000001abc",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.delete_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}
