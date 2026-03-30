package docusign

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateRecipients_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/test-account-id-456/envelopes/env-abc-123/recipients" {
			t.Errorf("expected path with recipients, got %s", got)
		}

		var body updateRecipientsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Signers) != 1 {
			t.Fatalf("expected 1 signer, got %d", len(body.Signers))
		}
		if body.Signers[0].RecipientID != "1" {
			t.Errorf("expected recipientId 1, got %q", body.Signers[0].RecipientID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updateRecipientsResponse{
			Signers: []recipientUpdateResult{
				{RecipientID: "1", RecipientIDGUID: "guid-1"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateRecipientsAction{conn: conn}

	params, _ := json.Marshal(updateRecipientsParams{
		EnvelopeID: "env-abc-123",
		Signers: []signerParam{
			{
				Email:        "new-signer@example.com",
				Name:         "New Signer",
				RecipientID:  "1",
				RoutingOrder: "1",
			},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.update_recipients",
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

	var signers []map[string]string
	if err := json.Unmarshal(data["signers"], &signers); err != nil {
		t.Fatalf("failed to unmarshal signers: %v", err)
	}
	if len(signers) != 1 {
		t.Fatalf("expected 1 signer result, got %d", len(signers))
	}
	if signers[0]["recipient_id"] != "1" {
		t.Errorf("expected recipient_id 1, got %q", signers[0]["recipient_id"])
	}
}

func TestUpdateRecipients_MissingEnvelopeID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecipientsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"signers": []map[string]string{
			{"email": "a@b.com", "name": "A", "recipient_id": "1", "routing_order": "1"},
		},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.update_recipients",
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

func TestUpdateRecipients_MissingSigners(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecipientsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"envelope_id": "env-abc-123",
		"signers":     []map[string]string{},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.update_recipients",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing signers")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateRecipients_IncompleteSignerFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateRecipientsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"envelope_id": "env-abc-123",
		"signers": []map[string]string{
			{"email": "a@b.com", "name": "A"},
		},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.update_recipients",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for incomplete signer fields")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
