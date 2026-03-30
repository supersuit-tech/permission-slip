package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateTags_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/tickets/42/tags.json" {
			t.Errorf("expected path /tickets/42/tags.json, got %s", r.URL.Path)
		}

		var body map[string][]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			return
		}
		if len(body["tags"]) != 2 {
			t.Errorf("expected 2 tags, got %d", len(body["tags"]))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tagsResponse{
			Tags: body["tags"],
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateTagsAction{conn: conn}

	params, _ := json.Marshal(updateTagsParams{
		TicketID: 42,
		Tags:     []string{"billing", "escalated"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_tags",
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

func TestUpdateTags_EmptyArrayClearsTags(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string][]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			return
		}
		if len(body["tags"]) != 0 {
			t.Errorf("expected empty tags array, got %d tags", len(body["tags"]))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tagsResponse{Tags: []string{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateTagsAction{conn: conn}

	params, _ := json.Marshal(updateTagsParams{
		TicketID: 42,
		Tags:     []string{},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_tags",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: empty tags should be allowed to clear all tags: %v", err)
	}
}

func TestUpdateTags_MissingTicketID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTagsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"tags": []string{"test"}})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_tags",
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

func TestUpdateTags_NilTagsArray(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTagsAction{conn: conn}

	// JSON object without tags field → Tags will be nil
	params, _ := json.Marshal(map[string]int64{"ticket_id": 42})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_tags",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for nil tags")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateTags_TooManyTags(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateTagsAction{conn: conn}

	tags := make([]string, 101)
	for i := range tags {
		tags[i] = "tag"
	}
	params, _ := json.Marshal(updateTagsParams{
		TicketID: 42,
		Tags:     tags,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.update_tags",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for too many tags")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
