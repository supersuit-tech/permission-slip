package confluence

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddAttachment_Success(t *testing.T) {
	t.Parallel()

	fileContent := []byte("fake PDF content")
	contentB64 := base64.StdEncoding.EncodeToString(fileContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/pages/123456/attachments" {
			t.Errorf("expected path /pages/123456/attachments, got %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Atlassian-Token"); got != "no-check" {
			t.Errorf("expected X-Atlassian-Token: no-check, got %q", got)
		}

		// Parse multipart body.
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("failed to parse content-type: %v", err)
		}
		if mediaType != "multipart/form-data" {
			t.Errorf("expected multipart/form-data, got %q", mediaType)
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		part, err := mr.NextPart()
		if err != nil {
			t.Fatalf("failed to read multipart part: %v", err)
		}
		if part.FormName() != "file" {
			t.Errorf("expected form field 'file', got %q", part.FormName())
		}
		if part.FileName() != "document.pdf" {
			t.Errorf("expected filename 'document.pdf', got %q", part.FileName())
		}
		body, _ := io.ReadAll(part)
		if string(body) != string(fileContent) {
			t.Errorf("file content mismatch")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(addAttachmentResponse{
			Results: []attachmentItem{
				{
					ID:        "att-new-1",
					Title:     "document.pdf",
					MediaType: "application/pdf",
					FileSize:  int64(len(fileContent)),
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &addAttachmentAction{conn: conn}

	params, _ := json.Marshal(addAttachmentParams{
		PageID:     "123456",
		Filename:   "document.pdf",
		ContentB64: contentB64,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.add_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "att-new-1" {
		t.Errorf("expected id 'att-new-1', got %v", data["id"])
	}
}

func TestAddAttachment_MissingPageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addAttachmentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"filename":        "doc.pdf",
		"content_base64":  base64.StdEncoding.EncodeToString([]byte("content")),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.add_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestAddAttachment_InvalidBase64(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addAttachmentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"page_id":        "123456",
		"filename":       "doc.pdf",
		"content_base64": "not valid base64!!!",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.add_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for invalid base64, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestAddAttachment_PathTraversalFilename(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addAttachmentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"page_id":        "123456",
		"filename":       "../etc/passwd",
		"content_base64": base64.StdEncoding.EncodeToString([]byte("content")),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.add_attachment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for path traversal filename, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
