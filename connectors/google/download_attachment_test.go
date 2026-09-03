package google

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func gmailAttachmentMessage(messageID, partID, attachmentID, filename, mimeType string, size int) gmailFullMessage {
	return gmailFullMessage{
		ID:       messageID,
		ThreadID: "thread-1",
		Payload: gmailMessagePart{
			MimeType: "multipart/mixed",
			Parts: []gmailMessagePart{
				{
					MimeType: "text/plain",
					Body: struct {
						AttachmentID string `json:"attachmentId"`
						Size         int    `json:"size"`
						Data         string `json:"data"`
					}{Data: base64.RawURLEncoding.EncodeToString([]byte("See attached."))},
				},
				{
					PartID:   partID,
					MimeType: mimeType,
					Filename: filename,
					Headers: []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					}{
						{Name: "Content-Disposition", Value: `attachment; filename="` + filename + `"`},
					},
					Body: struct {
						AttachmentID string `json:"attachmentId"`
						Size         int    `json:"size"`
						Data         string `json:"data"`
					}{
						AttachmentID: attachmentID,
						Size:         size,
					},
				},
			},
		},
	}
}

func TestDownloadAttachment_Success(t *testing.T) {
	t.Parallel()

	pdfBytes := []byte("%PDF-1.4 receipt")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/attachments/att-001"):
			json.NewEncoder(w).Encode(gmailAttachment{
				Size: len(pdfBytes),
				Data: base64.RawURLEncoding.EncodeToString(pdfBytes),
			})
		case strings.Contains(r.URL.Path, "/messages/msg-attach") && !strings.Contains(r.URL.Path, "/attachments/"):
			json.NewEncoder(w).Encode(gmailAttachmentMessage("msg-attach", "1", "att-001", "receipt.pdf", "application/pdf", len(pdfBytes)))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    "msg-attach",
		AttachmentID: "att-001",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.download_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data downloadAttachmentResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Filename != "receipt.pdf" {
		t.Errorf("expected filename receipt.pdf, got %q", data.Filename)
	}
	if data.MimeType != "application/pdf" {
		t.Errorf("expected mime_type application/pdf, got %q", data.MimeType)
	}
	if data.Size != len(pdfBytes) {
		t.Errorf("expected size %d, got %d", len(pdfBytes), data.Size)
	}
	if data.AttachmentID != "att-001" {
		t.Errorf("expected attachment_id att-001, got %q", data.AttachmentID)
	}

	decoded, err := base64.StdEncoding.DecodeString(data.ContentBase64)
	if err != nil {
		t.Fatalf("failed to decode content_base64: %v", err)
	}
	if string(decoded) != string(pdfBytes) {
		t.Errorf("expected decoded content %q, got %q", pdfBytes, decoded)
	}
}

func TestDownloadAttachment_MatchPartID(t *testing.T) {
	t.Parallel()

	pngBytes := []byte("\x89PNG\r\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/attachments/") {
			json.NewEncoder(w).Encode(gmailAttachment{
				Size: len(pngBytes),
				Data: base64.URLEncoding.EncodeToString(pngBytes),
			})
			return
		}
		json.NewEncoder(w).Encode(gmailAttachmentMessage("msg-2", "1.2", "ANGjdJ8-real", "scan.png", "image/png", len(pngBytes)))
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    "msg-2",
		AttachmentID: "1.2",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.download_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data downloadAttachmentResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.AttachmentID != "ANGjdJ8-real" {
		t.Errorf("expected canonical attachment_id ANGjdJ8-real, got %q", data.AttachmentID)
	}
	if data.Filename != "scan.png" {
		t.Errorf("expected filename scan.png, got %q", data.Filename)
	}

	decoded, err := base64.StdEncoding.DecodeString(data.ContentBase64)
	if err != nil {
		t.Fatalf("failed to decode content_base64: %v", err)
	}
	if string(decoded) != string(pngBytes) {
		t.Errorf("expected decoded PNG bytes, got %q", decoded)
	}
}

func TestDownloadAttachment_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailAttachmentMessage("msg-attach", "1", "att-001", "receipt.pdf", "application/pdf", 12))
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    "msg-attach",
		AttachmentID: "missing-id",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.download_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for unknown attachment")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDownloadAttachment_Oversized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailAttachmentMessage("msg-big", "1", "att-big", "huge.pdf", "application/pdf", maxAttachmentBytes+1))
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    "msg-big",
		AttachmentID: "att-big",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.download_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for oversized attachment")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Errorf("expected oversize message, got: %v", err)
	}
}

func TestDownloadAttachment_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    "msg-1",
		AttachmentID: "att-1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.download_attachment",
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

func TestDownloadAttachment_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &downloadAttachmentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.download_attachment",
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

func TestDownloadAttachment_MissingMessageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"attachment_id": "att-1"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.download_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDownloadAttachment_MissingAttachmentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"message_id": "msg-1"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.download_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing attachment_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDecodeBase64Bytes(t *testing.T) {
	t.Parallel()

	raw := []byte("hello-bytes")
	cases := []string{
		base64.StdEncoding.EncodeToString(raw),
		base64.RawStdEncoding.EncodeToString(raw),
		base64.URLEncoding.EncodeToString(raw),
		base64.RawURLEncoding.EncodeToString(raw),
	}
	for _, encoded := range cases {
		got, err := decodeBase64Bytes(encoded)
		if err != nil {
			t.Fatalf("decodeBase64Bytes(%q) error: %v", encoded, err)
		}
		if string(got) != string(raw) {
			t.Errorf("decodeBase64Bytes(%q) = %q, want %q", encoded, got, raw)
		}
	}

	if _, err := decodeBase64Bytes("not valid!!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
}
