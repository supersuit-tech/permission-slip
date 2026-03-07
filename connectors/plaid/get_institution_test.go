package plaid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetInstitution_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/institutions/get_by_id" {
			t.Errorf("path = %s, want /institutions/get_by_id", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		if body["institution_id"] != "ins_1" {
			t.Errorf("institution_id = %v, want ins_1", body["institution_id"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"institution": map[string]any{
				"institution_id": "ins_1",
				"name":           "Bank of America",
				"products":       []string{"auth", "balance", "transactions"},
				"country_codes":  []string{"US"},
			},
			"request_id": "req123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.get_institution"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_institution",
		Parameters:  json.RawMessage(`{"institution_id":"ins_1"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	inst, ok := data["institution"].(map[string]any)
	if !ok {
		t.Fatal("expected institution in response")
	}
	if inst["name"] != "Bank of America" {
		t.Errorf("name = %v, want Bank of America", inst["name"])
	}
}

func TestGetInstitution_WithCountryCodes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		codes, ok := body["country_codes"].([]any)
		if !ok || len(codes) != 2 {
			t.Errorf("expected 2 country codes, got %v", body["country_codes"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"institution": map[string]any{}, "request_id": "req"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.get_institution"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_institution",
		Parameters:  json.RawMessage(`{"institution_id":"ins_1","country_codes":["US","CA"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetInstitution_MissingInstitutionID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["plaid.get_institution"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.get_institution",
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
