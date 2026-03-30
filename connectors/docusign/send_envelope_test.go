package docusign

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendEnvelope_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/test-account-id-456/envelopes/env-abc-123" {
			t.Errorf("expected path with envelope ID, got %s", got)
		}

		var body sendEnvelopeRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Status != "sent" {
			t.Errorf("expected status sent, got %q", body.Status)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"envelopeId": "env-abc-123",
			"status":     "sent",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEnvelopeAction{conn: conn}

	params, _ := json.Marshal(sendEnvelopeParams{EnvelopeID: "env-abc-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.send_envelope",
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
	if data["envelope_id"] != "env-abc-123" {
		t.Errorf("expected envelope_id env-abc-123, got %q", data["envelope_id"])
	}
	if data["status"] != "sent" {
		t.Errorf("expected status sent, got %q", data["status"])
	}
}

func TestSendEnvelope_MissingEnvelopeID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEnvelopeAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.send_envelope",
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

func TestSendEnvelope_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"errorCode": "ENVELOPE_NOT_IN_CORRECT_STATE",
			"message":   "The envelope is not in a state that allows sending.",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEnvelopeAction{conn: conn}

	params, _ := json.Marshal(sendEnvelopeParams{EnvelopeID: "env-abc-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.send_envelope",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for API error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
