package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateRecord_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer 00Dxx0000000000!test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("expected Accept header 'application/json', got %q", got)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Lead/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["LastName"] != "Smith" {
			t.Errorf("expected LastName 'Smith', got %v", body["LastName"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sfCreateResponse{
			ID:      "00Qxx0000000001",
			Success: true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createRecordAction{conn: conn}

	params, _ := json.Marshal(createRecordParams{
		SObjectType: "Lead",
		Fields: map[string]any{
			"LastName": "Smith",
			"Company":  "Acme",
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_record",
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
	if data["id"] != "00Qxx0000000001" {
		t.Errorf("expected id '00Qxx0000000001', got %v", data["id"])
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

func TestCreateRecord_MissingSObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"fields": map[string]any{"Name": "Test"}})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_record",
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

func TestCreateRecord_MissingFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createRecordAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"sobject_type": "Lead"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_record",
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

func TestCreateRecord_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode([]sfAPIError{{ErrorCode: "INVALID_SESSION_ID", Message: "Session expired"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createRecordAction{conn: conn}

	params, _ := json.Marshal(createRecordParams{
		SObjectType: "Lead",
		Fields:      map[string]any{"LastName": "Test"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_record",
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

func TestCreateRecord_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode([]sfAPIError{{ErrorCode: "REQUEST_LIMIT_EXCEEDED", Message: "Too many requests"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createRecordAction{conn: conn}

	params, _ := json.Marshal(createRecordParams{
		SObjectType: "Lead",
		Fields:      map[string]any{"LastName": "Test"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T (%v)", err, err)
	}
}

func TestCreateRecord_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createRecordAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_record",
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

func TestCreateRecord_ValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode([]sfAPIError{{ErrorCode: "REQUIRED_FIELD_MISSING", Message: "Required field missing: LastName"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createRecordAction{conn: conn}

	params, _ := json.Marshal(createRecordParams{
		SObjectType: "Lead",
		Fields:      map[string]any{"Company": "Acme"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_record",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for validation failure")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T (%v)", err, err)
	}
}
