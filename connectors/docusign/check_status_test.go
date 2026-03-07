package docusign

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckStatus_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")

		// Route based on path: envelope vs recipients endpoint
		if strings.HasSuffix(r.URL.Path, "/recipients") {
			json.NewEncoder(w).Encode(map[string]any{
				"signers": []map[string]string{
					{
						"recipientId":    "1",
						"name":           "Jane Doe",
						"email":          "jane@example.com",
						"status":         "completed",
						"routingOrder":   "1",
						"signedDateTime": "2026-03-07T11:30:00Z",
					},
				},
			})
			return
		}

		if got := r.URL.Path; got != "/accounts/test-account-id-456/envelopes/env-abc-123" {
			t.Errorf("expected path with envelope ID, got %s", got)
		}
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

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	var status string
	json.Unmarshal(data["status"], &status)
	if status != "completed" {
		t.Errorf("expected status completed, got %q", status)
	}

	var completedDate string
	json.Unmarshal(data["completed_date"], &completedDate)
	if completedDate != "2026-03-07T12:00:00Z" {
		t.Errorf("expected completed_date, got %q", completedDate)
	}

	// Verify recipients are included by default
	if _, ok := data["recipients"]; !ok {
		t.Error("expected recipients to be included by default")
	}

	var recipients []map[string]string
	if err := json.Unmarshal(data["recipients"], &recipients); err != nil {
		t.Fatalf("failed to unmarshal recipients: %v", err)
	}
	if len(recipients) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(recipients))
	}
	if recipients[0]["name"] != "Jane Doe" {
		t.Errorf("expected recipient name Jane Doe, got %q", recipients[0]["name"])
	}
	if recipients[0]["status"] != "completed" {
		t.Errorf("expected recipient status completed, got %q", recipients[0]["status"])
	}
}

func TestCheckStatus_WithoutRecipients(t *testing.T) {
	t.Parallel()

	requestPaths := make([]string, 0)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPaths = append(requestPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"envelopeId":            "env-abc-123",
			"status":                "sent",
			"statusChangedDateTime": "2026-03-07T10:00:00Z",
			"emailSubject":          "Please sign",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &checkStatusAction{conn: conn}

	includeRecipients := false
	params, _ := json.Marshal(checkStatusParams{
		EnvelopeID:        "env-abc-123",
		IncludeRecipients: &includeRecipients,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.check_status",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only hit the envelope endpoint, not recipients
	if len(requestPaths) != 1 {
		t.Errorf("expected 1 API call (envelope only), got %d", len(requestPaths))
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if _, ok := data["recipients"]; ok {
		t.Error("expected recipients to be excluded when include_recipients=false")
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/recipients") {
			json.NewEncoder(w).Encode(map[string]any{"signers": []any{}})
			return
		}
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

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	var status string
	json.Unmarshal(data["status"], &status)
	if status != "voided" {
		t.Errorf("expected status voided, got %q", status)
	}

	var voidedReason string
	json.Unmarshal(data["voided_reason"], &voidedReason)
	if voidedReason != "No longer needed" {
		t.Errorf("expected voided_reason, got %q", voidedReason)
	}
}
