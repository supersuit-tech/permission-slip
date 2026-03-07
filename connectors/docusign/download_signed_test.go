package docusign

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDownloadSigned_Success(t *testing.T) {
	t.Parallel()

	pdfContent := []byte("%PDF-1.4 fake pdf content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/accounts/test-account-id-456/envelopes/env-abc-123/documents/combined" {
			t.Errorf("expected path with documents/combined, got %s", got)
		}
		if got := r.Header.Get("Accept"); got != "application/pdf" {
			t.Errorf("expected Accept: application/pdf, got %q", got)
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Write(pdfContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &downloadSignedAction{conn: conn}

	params, _ := json.Marshal(downloadSignedParams{EnvelopeID: "env-abc-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.download_signed",
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
	if data["document_id"] != "combined" {
		t.Errorf("expected document_id combined, got %q", data["document_id"])
	}
	if data["encoding"] != "base64" {
		t.Errorf("expected encoding base64, got %q", data["encoding"])
	}
	if data["mime_type"] != "application/pdf" {
		t.Errorf("expected mime_type application/pdf, got %q", data["mime_type"])
	}

	decoded, err := base64.StdEncoding.DecodeString(data["content"])
	if err != nil {
		t.Fatalf("failed to decode base64 content: %v", err)
	}
	if string(decoded) != string(pdfContent) {
		t.Errorf("expected decoded content to match PDF, got %q", string(decoded))
	}
}

func TestDownloadSigned_SpecificDocument(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/accounts/test-account-id-456/envelopes/env-abc-123/documents/doc-1" {
			t.Errorf("expected path with documents/doc-1, got %s", got)
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("pdf"))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &downloadSignedAction{conn: conn}

	params, _ := json.Marshal(downloadSignedParams{
		EnvelopeID: "env-abc-123",
		DocumentID: "doc-1",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.download_signed",
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
	if data["document_id"] != "doc-1" {
		t.Errorf("expected document_id doc-1, got %q", data["document_id"])
	}
}

func TestDownloadSigned_MissingEnvelopeID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &downloadSignedAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.download_signed",
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
