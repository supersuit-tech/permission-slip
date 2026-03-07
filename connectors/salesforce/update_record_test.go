package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateRecord_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Lead/00Qxx0000000001" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["Status"] != "Closed" {
			t.Errorf("expected Status 'Closed', got %v", body["Status"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateRecordAction{conn: conn}

	params, _ := json.Marshal(updateRecordParams{
		SObjectType: "Lead",
		RecordID:    "00Qxx0000000001",
		Fields:      map[string]any{"Status": "Closed"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_record",
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
	if data["record_id"] != "00Qxx0000000001" {
		t.Errorf("expected record_id '00Qxx0000000001', got %v", data["record_id"])
	}
	if data["sobject_type"] != "Lead" {
		t.Errorf("expected sobject_type 'Lead', got %v", data["sobject_type"])
	}
	if data["success"] != true {
		t.Errorf("expected success true, got %v", data["success"])
	}
	if data["record_url"] != "https://myorg.salesforce.com/00Qxx0000000001" {
		t.Errorf("expected record_url, got %v", data["record_url"])
	}
}

func TestUpdateRecord_MissingRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"sobject_type": "Lead",
		"fields":       map[string]any{"Status": "Closed"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_record",
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

func TestUpdateRecord_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode([]sfAPIError{{ErrorCode: "INVALID_SESSION_ID", Message: "Session expired"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateRecordAction{conn: conn}

	params, _ := json.Marshal(updateRecordParams{
		SObjectType: "Lead",
		RecordID:    "00Qxx0000000001",
		Fields:      map[string]any{"Status": "Closed"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestUpdateRecord_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_record",
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

func TestUpdateRecord_MissingSObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"record_id": "00Qxx0000000001",
		"fields":    map[string]any{"Status": "Closed"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_record",
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

func TestUpdateRecord_MissingFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"sobject_type": "Lead",
		"record_id":    "00Qxx0000000001",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateRecord_InvalidRecordID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"sobject_type": "Lead",
		"record_id":    "abc",
		"fields":       map[string]any{"Status": "Closed"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.update_record",
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
