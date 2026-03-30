package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListTags_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/tickets/42/tags.json" {
			t.Errorf("expected path /tickets/42/tags.json, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tagsResponse{
			Tags: []string{"billing", "vip"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listTagsAction{conn: conn}

	params, _ := json.Marshal(listTagsParams{TicketID: 42})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.list_tags",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data tagsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(data.Tags))
	}
}

func TestListTags_MissingTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listTagsAction{conn: conn}

	params, _ := json.Marshal(map[string]int64{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.list_tags",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing ticket_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
