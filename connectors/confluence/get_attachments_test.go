package confluence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetAttachments_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/pages/123456/attachments" {
			t.Errorf("expected path /pages/123456/attachments, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(getAttachmentsResponse{
			Results: []attachmentItem{
				{
					ID:        "att-1",
					Title:     "diagram.png",
					MediaType: "image/png",
					FileSize:  204800,
				},
				{
					ID:        "att-2",
					Title:     "spec.pdf",
					MediaType: "application/pdf",
					FileSize:  1048576,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getAttachmentsAction{conn: conn}

	params, _ := json.Marshal(getAttachmentsParams{PageID: "123456"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.get_attachments",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data getAttachmentsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Results) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(data.Results))
	}
	if data.Results[0].Title != "diagram.png" {
		t.Errorf("expected first attachment 'diagram.png', got %q", data.Results[0].Title)
	}
}

func TestGetAttachments_MissingPageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getAttachmentsAction{conn: conn}

	params, _ := json.Marshal(map[string]int{"limit": 10})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.get_attachments",
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
