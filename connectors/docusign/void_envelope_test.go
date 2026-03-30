package docusign

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestVoidEnvelope_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/test-account-id-456/envelopes/env-abc-123" {
			t.Errorf("expected path with envelope ID, got %s", got)
		}

		var body voidEnvelopeRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Status != "voided" {
			t.Errorf("expected status voided, got %q", body.Status)
		}
		if body.VoidedReason != "Contract terms changed" {
			t.Errorf("expected voided reason, got %q", body.VoidedReason)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &voidEnvelopeAction{conn: conn}

	params, _ := json.Marshal(voidEnvelopeParams{
		EnvelopeID: "env-abc-123",
		VoidReason: "Contract terms changed",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.void_envelope",
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
}

func TestVoidEnvelope_MissingEnvelopeID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &voidEnvelopeAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"void_reason": "Changed my mind",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.void_envelope",
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

func TestVoidEnvelope_MissingVoidReason(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &voidEnvelopeAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"envelope_id": "env-abc-123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.void_envelope",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing void_reason")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
