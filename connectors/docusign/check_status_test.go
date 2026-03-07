package docusign

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckStatus_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/test-account-id-456/envelopes/env-abc-123" {
			t.Errorf("expected path with envelope ID, got %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"envelopeId":            "env-abc-123",
			"status":                "completed",
			"statusChangedDateTime": "2026-03-07T12:00:00Z",
			"emailSubject":          "Please sign this",
			"sentDateTime":          "2026-03-07T10:00:00Z",
			"completedDateTime":     "2026-03-07T12:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &checkStatusAction{conn: conn}

	params, _ := json.Marshal(checkStatusParams{EnvelopeID: "env-abc-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.check_status",
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
	if data["status"] != "completed" {
		t.Errorf("expected status completed, got %q", data["status"])
	}
	if data["completed_date"] != "2026-03-07T12:00:00Z" {
		t.Errorf("expected completed_date, got %q", data["completed_date"])
	}
	if data["sent_date"] != "2026-03-07T10:00:00Z" {
		t.Errorf("expected sent_date, got %q", data["sent_date"])
	}
}

func TestCheckStatus_MissingEnvelopeID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &checkStatusAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.check_status",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing envelope_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckStatus_VoidedEnvelope(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"envelopeId":            "env-voided",
			"status":                "voided",
			"statusChangedDateTime": "2026-03-07T14:00:00Z",
			"emailSubject":          "Voided doc",
			"voidedDateTime":        "2026-03-07T14:00:00Z",
			"voidedReason":          "No longer needed",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &checkStatusAction{conn: conn}

	params, _ := json.Marshal(checkStatusParams{EnvelopeID: "env-voided"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.check_status",
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
	if data["status"] != "voided" {
		t.Errorf("expected status voided, got %q", data["status"])
	}
	if data["voided_reason"] != "No longer needed" {
		t.Errorf("expected voided_reason, got %q", data["voided_reason"])
	}
}
