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
	if att.AttachmentID != "att-001" {
		t.Errorf("expected attachment_id 'att-001', got %q", att.AttachmentID)
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

func TestReadEmail_FormatMetadata(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("format") != "metadata" {
			t.Errorf("expected format=metadata, got %s", r.URL.Query().Get("format"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailFullMessage{
			ID:       "msg-meta",
			ThreadID: "thread-meta",
			LabelIDs: []string{"INBOX"},
			Snippet:  "Hello snippet",
			Payload: gmailMessagePart{
				MimeType: "text/plain",
				Headers: []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				}{
					{Name: "From", Value: "meta@example.com"},
					{Name: "Subject", Value: "Metadata Only"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "msg-meta", Format: "metadata"})

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

	if detail.ID != "msg-meta" {
		t.Errorf("expected ID msg-meta, got %s", detail.ID)
	}
	if len(detail.Labels) != 1 || detail.Labels[0] != "INBOX" {
		t.Errorf("expected Labels [INBOX], got %v", detail.Labels)
	}
	if detail.From != "meta@example.com" {
		t.Errorf("expected From meta@example.com, got %s", detail.From)
	}
	if detail.Subject != "Metadata Only" {
		t.Errorf("expected Subject 'Metadata Only', got %s", detail.Subject)
	}
	if detail.Snippet != "Hello snippet" {
		t.Errorf("expected Snippet 'Hello snippet', got %q", detail.Snippet)
	}
	if detail.Body != "" {
		t.Errorf("expected empty Body for format=metadata, got %q", detail.Body)
	}
	if detail.ContentType != "" {
		t.Errorf("expected empty ContentType for format=metadata, got %q", detail.ContentType)
	}
}

func TestReadEmail_FormatMinimal(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "minimal" {
			t.Errorf("expected format=minimal, got %s", r.URL.Query().Get("format"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailFullMessage{
			ID:       "msg-min",
			ThreadID: "thread-min",
			LabelIDs: []string{"INBOX", "UNREAD"},
			Snippet:  "Short snippet",
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "msg-min", Format: "minimal"})

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

	if detail.ID != "msg-min" {
		t.Errorf("expected ID msg-min, got %s", detail.ID)
	}
	if detail.ThreadID != "thread-min" {
		t.Errorf("expected ThreadID thread-min, got %s", detail.ThreadID)
	}
	if len(detail.Labels) != 2 || detail.Labels[0] != "INBOX" || detail.Labels[1] != "UNREAD" {
		t.Errorf("expected Labels [INBOX UNREAD], got %v", detail.Labels)
	}
	if detail.Snippet != "Short snippet" {
		t.Errorf("expected Snippet 'Short snippet', got %q", detail.Snippet)
	}
	if detail.Body != "" {
		t.Errorf("expected empty Body for format=minimal, got %q", detail.Body)
	}
	if detail.ContentType != "" {
		t.Errorf("expected empty ContentType for format=minimal, got %q", detail.ContentType)
	}
}

func TestReadEmail_DefaultFormatIsFull(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("format") != "full" {
			t.Errorf("expected format=full (default), got %s", r.URL.Query().Get("format"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailFullMessage{
			ID:       "msg-default",
			ThreadID: "thread-default",
		})
	}))
	defer srv.Close()

	conn := newGmailForTest(srv.Client(), srv.URL)
	action := &readEmailAction{conn: conn}

	// No format specified — should default to "full".
	params, _ := json.Marshal(readEmailParams{MessageID: "msg-default"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadEmail_InvalidFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readEmailAction{conn: conn}

	params, _ := json.Marshal(readEmailParams{MessageID: "msg-123", Format: "raw"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.read_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
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
		{`attachment; filename*=UTF-8''caf%C3%A9%20menu.pdf`, "café menu.pdf"},
		{`attachment; filename="fallback.pdf"; filename*=UTF-8''preferred.pdf`, "preferred.pdf"},
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

func TestDecodeRFC5987_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid UTF-8", "UTF-8''hello%20world.pdf", "hello world.pdf"},
		{"empty language tag", "UTF-8'en'doc.pdf", "doc.pdf"},
		{"missing single quotes", "UTF-8hello.pdf", ""},
		{"only one single quote", "UTF-8'hello.pdf", ""},
		{"invalid percent encoding", "UTF-8''bad%ZZvalue", ""},
		{"empty value after quotes", "UTF-8''", ""},
		{"empty input", "", ""},
		{"non-UTF-8 charset rejected", "ISO-8859-1''caf%E9.pdf", ""},
	}

	for _, tt := range tests {
		got := decodeRFC5987(tt.input)
		if got != tt.want {
			t.Errorf("decodeRFC5987[%s](%q) = %q, want %q", tt.name, tt.input, got, tt.want)
		}
	}
}

func TestDecodeBase64URL_EdgeCases(t *testing.T) {
	t.Parallel()

	// Raw base64url (no padding) — standard Gmail format.
	raw := base64.RawURLEncoding.EncodeToString([]byte("test data"))
	if got := decodeBase64URL(raw); got != "test data" {
		t.Errorf("raw base64url: got %q, want %q", got, "test data")
	}

	// Padded base64url — fallback path.
	padded := base64.URLEncoding.EncodeToString([]byte("padded data"))
	if got := decodeBase64URL(padded); got != "padded data" {
		t.Errorf("padded base64url: got %q, want %q", got, "padded data")
	}

	// Totally invalid base64 — returned as-is.
	invalid := "not-valid-base64!!!"
	if got := decodeBase64URL(invalid); got != invalid {
		t.Errorf("invalid base64: got %q, want %q (as-is)", got, invalid)
	}
}

func TestTruncateEmailBody_UTF8Boundary(t *testing.T) {
	t.Parallel()

	// Short body — no truncation.
	short := "hello"
	if got := truncateEmailBody(short); got != short {
		t.Errorf("short body: got %q, want %q", got, short)
	}

	// Empty body.
	if got := truncateEmailBody(""); got != "" {
		t.Errorf("empty body: got %q, want empty", got)
	}
}

func TestExtractBody_DepthLimit(t *testing.T) {
	t.Parallel()

	plainData := base64.RawURLEncoding.EncodeToString([]byte("deep body"))

	// Create a part at exactly the depth limit — should still return empty
	// because the body is nested beyond maxMIMEDepth.
	part := &gmailMessagePart{
		MimeType: "multipart/mixed",
		Parts: []gmailMessagePart{
			{
				MimeType: "text/plain",
				Body: struct {
					AttachmentID string `json:"attachmentId"`
					Size         int    `json:"size"`
					Data         string `json:"data"`
				}{Data: plainData},
			},
		},
	}

	// At depth maxMIMEDepth+1, extractBody should bail out.
	body, ct := extractBody(part, maxMIMEDepth+1)
	if body != "" {
		t.Errorf("expected empty body at max depth, got %q", body)
	}
	if ct != "text/plain" {
		t.Errorf("expected text/plain fallback, got %s", ct)
	}
}
