package google

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestReadEmail_Success(t *testing.T) {
	t.Parallel()

	plainBody := base64.RawURLEncoding.EncodeToString([]byte("Hello, this is the email body."))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("format") != "full" {
			t.Errorf("expected format=full, got %s", r.URL.Query().Get("format"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailFullMessage{
			ID:       "msg-123",
			ThreadID: "thread-456",
			LabelIDs: []string{"INBOX", "UNREAD"},
			Snippet:  "Hello, this is",
			Payload: gmailMessagePart{
				MimeType: "text/plain",
				Headers: []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				}{
					{Name: "From", Value: "sender@example.com"},
					{Name: "To", Value: "recipient@example.com"},
					{Name: "Subject", Value: "Test Email"},
					{Name: "Date", Value: "Mon, 14 Mar 2026 10:00:00 -0500"},
				},
				Body: struct {
					AttachmentID string `json:"attachmentId"`
					Size         int    `json:"size"`
					Data         string `json:"data"`
				}{
					Data: plainBody,
					Size: 29,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "msg-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var detail emailFullDetail
	if err := json.Unmarshal(result.Data, &detail); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if detail.ID != "msg-123" {
		t.Errorf("expected ID msg-123, got %s", detail.ID)
	}
	if detail.ThreadID != "thread-456" {
		t.Errorf("expected ThreadID thread-456, got %s", detail.ThreadID)
	}
	if detail.From != "sender@example.com" {
		t.Errorf("expected From sender@example.com, got %s", detail.From)
	}
	if detail.Subject != "Test Email" {
		t.Errorf("expected Subject 'Test Email', got %s", detail.Subject)
	}
	if detail.Body != "Hello, this is the email body." {
		t.Errorf("expected body content, got %q", detail.Body)
	}
	if detail.ContentType != "text/plain" {
		t.Errorf("expected content_type text/plain, got %s", detail.ContentType)
	}
	if detail.Snippet != "Hello, this is" {
		t.Errorf("expected snippet 'Hello, this is', got %q", detail.Snippet)
	}
}

func TestReadEmail_MultipartMessage(t *testing.T) {
	t.Parallel()

	plainBody := base64.RawURLEncoding.EncodeToString([]byte("Plain text version"))
	htmlBody := base64.RawURLEncoding.EncodeToString([]byte("<p>HTML version</p>"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailFullMessage{
			ID:       "msg-multi",
			ThreadID: "thread-1",
			Payload: gmailMessagePart{
				MimeType: "multipart/alternative",
				Parts: []gmailMessagePart{
					{
						MimeType: "text/plain",
						Body: struct {
							AttachmentID string `json:"attachmentId"`
							Size         int    `json:"size"`
							Data         string `json:"data"`
						}{Data: plainBody},
					},
					{
						MimeType: "text/html",
						Body: struct {
							AttachmentID string `json:"attachmentId"`
							Size         int    `json:"size"`
							Data         string `json:"data"`
						}{Data: htmlBody},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "msg-multi"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var detail emailFullDetail
	if err := json.Unmarshal(result.Data, &detail); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	// Should prefer text/plain over text/html.
	if detail.ContentType != "text/plain" {
		t.Errorf("expected text/plain, got %s", detail.ContentType)
	}
	if detail.Body != "Plain text version" {
		t.Errorf("expected plain text body, got %q", detail.Body)
	}
}

func TestReadEmail_WithAttachments(t *testing.T) {
	t.Parallel()

	plainBody := base64.RawURLEncoding.EncodeToString([]byte("See attached."))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailFullMessage{
			ID:       "msg-attach",
			ThreadID: "thread-2",
			Payload: gmailMessagePart{
				MimeType: "multipart/mixed",
				Parts: []gmailMessagePart{
					{
						MimeType: "text/plain",
						Body: struct {
							AttachmentID string `json:"attachmentId"`
							Size         int    `json:"size"`
							Data         string `json:"data"`
						}{Data: plainBody},
					},
					{
						PartID:   "1",
						MimeType: "application/pdf",
						Headers: []struct {
							Name  string `json:"name"`
							Value string `json:"value"`
						}{
							{Name: "Content-Disposition", Value: `attachment; filename="report.pdf"`},
						},
						Body: struct {
							AttachmentID string `json:"attachmentId"`
							Size         int    `json:"size"`
							Data         string `json:"data"`
						}{
							AttachmentID: "att-001",
							Size:         12345,
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "msg-attach"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var detail emailFullDetail
	if err := json.Unmarshal(result.Data, &detail); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(detail.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(detail.Attachments))
	}
	att := detail.Attachments[0]
	if att.Filename != "report.pdf" {
		t.Errorf("expected filename report.pdf, got %s", att.Filename)
	}
	if att.MimeType != "application/pdf" {
		t.Errorf("expected mime_type application/pdf, got %s", att.MimeType)
	}
	if att.Size != 12345 {
		t.Errorf("expected size 12345, got %d", att.Size)
	}
	if att.PartID != "1" {
		t.Errorf("expected part_id '1', got %q", att.PartID)
	}
}

func TestReadEmail_AttachmentFilenameFromContentType(t *testing.T) {
	t.Parallel()

	plainBody := base64.RawURLEncoding.EncodeToString([]byte("See attached."))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailFullMessage{
			ID:       "msg-ct-name",
			ThreadID: "thread-3",
			Payload: gmailMessagePart{
				MimeType: "multipart/mixed",
				Parts: []gmailMessagePart{
					{
						MimeType: "text/plain",
						Body: struct {
							AttachmentID string `json:"attachmentId"`
							Size         int    `json:"size"`
							Data         string `json:"data"`
						}{Data: plainBody},
					},
					{
						MimeType: "application/pdf",
						Headers: []struct {
							Name  string `json:"name"`
							Value string `json:"value"`
						}{
							// No Content-Disposition; filename in Content-Type name= instead.
							{Name: "Content-Type", Value: `application/pdf; name="invoice.pdf"`},
						},
						Body: struct {
							AttachmentID string `json:"attachmentId"`
							Size         int    `json:"size"`
							Data         string `json:"data"`
						}{
							AttachmentID: "att-002",
							Size:         5000,
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "msg-ct-name"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var detail emailFullDetail
	if err := json.Unmarshal(result.Data, &detail); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if len(detail.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(detail.Attachments))
	}
	if detail.Attachments[0].Filename != "invoice.pdf" {
		t.Errorf("expected filename 'invoice.pdf' from Content-Type fallback, got %q", detail.Attachments[0].Filename)
	}
}

func TestReadEmail_MissingMessageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
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

func TestReadEmail_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readEmailAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
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

func TestReadEmail_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "msg-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
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

func TestReadEmail_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 404, "message": "Requested entity was not found."},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "nonexistent"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestExtractBody_PlainTextPreferred(t *testing.T) {
	t.Parallel()

	plainData := base64.RawURLEncoding.EncodeToString([]byte("plain"))
	htmlData := base64.RawURLEncoding.EncodeToString([]byte("<b>html</b>"))

	part := &gmailMessagePart{
		MimeType: "multipart/alternative",
		Parts: []gmailMessagePart{
			{MimeType: "text/html", Body: struct {
				AttachmentID string `json:"attachmentId"`
				Size         int    `json:"size"`
				Data         string `json:"data"`
			}{Data: htmlData}},
			{MimeType: "text/plain", Body: struct {
				AttachmentID string `json:"attachmentId"`
				Size         int    `json:"size"`
				Data         string `json:"data"`
			}{Data: plainData}},
		},
	}

	body, ct := extractBody(part, 0)
	if ct != "text/plain" {
		t.Errorf("expected text/plain, got %s", ct)
	}
	if body != "plain" {
		t.Errorf("expected 'plain', got %q", body)
	}
}

func TestExtractBody_FallbackToHTML(t *testing.T) {
	t.Parallel()

	htmlData := base64.RawURLEncoding.EncodeToString([]byte("<b>html only</b>"))

	part := &gmailMessagePart{
		MimeType: "multipart/alternative",
		Parts: []gmailMessagePart{
			{MimeType: "text/html", Body: struct {
				AttachmentID string `json:"attachmentId"`
				Size         int    `json:"size"`
				Data         string `json:"data"`
			}{Data: htmlData}},
		},
	}

	body, ct := extractBody(part, 0)
	if ct != "text/html" {
		t.Errorf("expected text/html, got %s", ct)
	}
	if body != "<b>html only</b>" {
		t.Errorf("expected html body, got %q", body)
	}
}

func TestParseFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{`attachment; filename="report.pdf"`, "report.pdf"},
		{`attachment; filename=report.pdf`, "report.pdf"},
		{`application/pdf; name="invoice.pdf"`, "invoice.pdf"},
		{`application/pdf; name=invoice.pdf`, "invoice.pdf"},
		{`inline`, ""},
		{``, ""},
	}

	for _, tt := range tests {
		got := parseFilename(tt.input)
		if got != tt.want {
			t.Errorf("parseFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
