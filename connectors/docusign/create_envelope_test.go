package docusign

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateEnvelope_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/test-account-id-456/envelopes" {
			t.Errorf("expected path /accounts/test-account-id-456/envelopes, got %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		var body createEnvelopeRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.TemplateID != "tpl-123" {
			t.Errorf("expected templateId tpl-123, got %q", body.TemplateID)
		}
		if body.Status != "created" {
			t.Errorf("expected status created, got %q", body.Status)
		}
		if len(body.TemplateRoles) != 1 {
			t.Fatalf("expected 1 template role, got %d", len(body.TemplateRoles))
		}
		if body.TemplateRoles[0].Email != "signer@example.com" {
			t.Errorf("expected email signer@example.com, got %q", body.TemplateRoles[0].Email)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"envelopeId":     "env-abc-123",
			"status":         "created",
			"uri":            "/envelopes/env-abc-123",
			"statusDateTime": "2026-03-07T10:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createEnvelopeAction{conn: conn}

	params, _ := json.Marshal(createEnvelopeParams{
		TemplateID:   "tpl-123",
		EmailSubject: "Please sign this",
		Recipients: []envelopeRecipient{
			{Email: "signer@example.com", Name: "Jane Doe", RoleName: "Signer"},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.create_envelope",
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
	if data["status"] != "created" {
		t.Errorf("expected status created, got %q", data["status"])
	}
}

func TestCreateEnvelope_MissingTemplateID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createEnvelopeAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"recipients": []map[string]string{
			{"email": "a@b.com", "name": "A", "role_name": "Signer"},
		},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.create_envelope",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing template_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateEnvelope_MissingRecipients(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createEnvelopeAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"template_id": "tpl-123",
		"recipients":  []map[string]string{},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.create_envelope",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing recipients")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateEnvelope_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"errorCode": "AUTHORIZATION_INVALID_TOKEN",
			"message":   "The access token provided is expired, revoked or malformed.",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createEnvelopeAction{conn: conn}

	params, _ := json.Marshal(createEnvelopeParams{
		TemplateID: "tpl-123",
		Recipients: []envelopeRecipient{
			{Email: "a@b.com", Name: "A", RoleName: "Signer"},
		},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.create_envelope",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCreateEnvelope_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createEnvelopeAction{conn: conn}

	params, _ := json.Marshal(createEnvelopeParams{
		TemplateID: "tpl-123",
		Recipients: []envelopeRecipient{
			{Email: "a@b.com", Name: "A", RoleName: "Signer"},
		},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.create_envelope",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestCreateEnvelope_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createEnvelopeAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.create_envelope",
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
