package protonmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDownloadAttachment_Success(t *testing.T) {
	t.Parallel()

	old := fetchAttachmentContent
	fetchAttachmentContent = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ map[string]uint32, folder string, messageID uint32, partPath []int) (*fetchedAttachment, error) {
		if folder != "INBOX" || messageID != 42 || formatPartPath(partPath) != "2.1" {
			return nil, fmt.Errorf("unexpected fetch args: folder=%q messageID=%d path=%v", folder, messageID, partPath)
		}
		return &fetchedAttachment{
			Filename:    "report.pdf",
			ContentType: "application/pdf",
			Content:     []byte("pdf-bytes"),
		}, nil
	}
	t.Cleanup(func() { fetchAttachmentContent = old })

	conn := New()
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    42,
		Folder:       "INBOX",
		AttachmentID: "2.1",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.download_attachment",
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
	if data["filename"] != "report.pdf" {
		t.Errorf("expected filename report.pdf, got %v", data["filename"])
	}
	if data["content_type"] != "application/pdf" {
		t.Errorf("expected content_type application/pdf, got %v", data["content_type"])
	}
	if int(data["size"].(float64)) != len("pdf-bytes") {
		t.Errorf("expected size %d, got %v", len("pdf-bytes"), data["size"])
	}

	decoded, err := base64.StdEncoding.DecodeString(data["content_base64"].(string))
	if err != nil {
		t.Fatalf("failed to decode content_base64: %v", err)
	}
	if string(decoded) != "pdf-bytes" {
		t.Errorf("expected decoded content %q, got %q", "pdf-bytes", string(decoded))
	}
}

func TestDownloadAttachment_MessageNotFound(t *testing.T) {
	t.Parallel()

	old := fetchAttachmentContent
	fetchAttachmentContent = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ map[string]uint32, _ string, _ uint32, _ []int) (*fetchedAttachment, error) {
		return nil, &connectors.ValidationError{Message: `message uid 99 not found in folder "INBOX"`}
	}
	t.Cleanup(func() { fetchAttachmentContent = old })

	conn := New()
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    99,
		AttachmentID: "2.1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.download_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for unknown message")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDownloadAttachment_AttachmentNotFound(t *testing.T) {
	t.Parallel()

	old := fetchAttachmentContent
	fetchAttachmentContent = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ map[string]uint32, _ string, _ uint32, _ []int) (*fetchedAttachment, error) {
		return nil, &connectors.ValidationError{Message: `attachment "9.9" not found on message uid 42 in folder "INBOX"`}
	}
	t.Cleanup(func() { fetchAttachmentContent = old })

	conn := New()
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    42,
		AttachmentID: "9.9",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.download_attachment",
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

	old := fetchAttachmentContent
	fetchAttachmentContent = func(_ context.Context, _ *ProtonMailConnector, _ connectors.Credentials, _ map[string]uint32, _ string, _ uint32, _ []int) (*fetchedAttachment, error) {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("attachment %q exceeds maximum size of %d bytes", "2.1", maxBodySize),
		}
	}
	t.Cleanup(func() { fetchAttachmentContent = old })

	conn := New()
	action := &downloadAttachmentAction{conn: conn}

	params, _ := json.Marshal(downloadAttachmentParams{
		MessageID:    42,
		AttachmentID: "2.1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.download_attachment",
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

func TestDownloadAttachment_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &downloadAttachmentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.download_attachment",
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

	params, _ := json.Marshal(map[string]any{
		"attachment_id": "2.1",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.download_attachment",
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

	params, _ := json.Marshal(map[string]any{
		"message_id": 42,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.download_attachment",
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

func TestDownloadAttachmentParams_Defaults(t *testing.T) {
	t.Parallel()

	p := &downloadAttachmentParams{
		MessageID:    1,
		AttachmentID: "2.1",
	}
	if err := p.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Folder != "INBOX" {
		t.Errorf("expected default folder INBOX, got %q", p.Folder)
	}
}

func TestParsePartPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    []int
		wantErr bool
	}{
		{"2.1", []int{2, 1}, false},
		{"1", []int{1}, false},
		{"", nil, true},
		{"0.1", nil, true},
		{"2.a", nil, true},
		{"2..1", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parsePartPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePartPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && !pathsEqual(got, tt.want) {
				t.Errorf("parsePartPath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatPartPath(t *testing.T) {
	t.Parallel()

	if got := formatPartPath([]int{2, 1}); got != "2.1" {
		t.Errorf("formatPartPath([]int{2,1}) = %q, want 2.1", got)
	}
	if got := formatPartPath(nil); got != "" {
		t.Errorf("formatPartPath(nil) = %q, want empty", got)
	}
}

func TestDecodeTransferEncoding(t *testing.T) {
	t.Parallel()

	raw := []byte("SGVsbG8=")
	decoded, err := decodeTransferEncoding(raw, "base64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(decoded) != "Hello" {
		t.Errorf("expected Hello, got %q", string(decoded))
	}

	plain, err := decodeTransferEncoding([]byte("plain"), "7bit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(plain) != "plain" {
		t.Errorf("expected plain, got %q", string(plain))
	}

	qpRaw := []byte("Hello=20World")
	qpDecoded, err := decodeTransferEncoding(qpRaw, "quoted-printable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(qpDecoded) != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", string(qpDecoded))
	}
}
