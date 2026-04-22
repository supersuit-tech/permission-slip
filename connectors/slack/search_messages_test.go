package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchMessages_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/search.messages" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body searchMessagesRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Query != "deploy in:#engineering" {
			t.Errorf("expected query 'deploy in:#engineering', got %q", body.Query)
		}
		if body.Count != 20 {
			t.Errorf("expected count 20, got %d", body.Count)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"messages": map[string]any{
				"matches": []map[string]any{
					{
						"channel":   map[string]string{"id": "C001", "name": "engineering"},
						"user":      "U001",
						"username":  "jdoe",
						"text":      "Deploying v2.0 now",
						"ts":        "1234567890.123456",
						"permalink": "https://team.slack.com/archives/C001/p1234567890123456",
					},
					{
						"channel":  map[string]string{"id": "C001", "name": "engineering"},
						"user":     "U002",
						"username": "asmith",
						"text":     "Deploy complete",
						"ts":       "1234567891.123456",
					},
				},
				"paging": map[string]int{
					"count": 20,
					"total": 2,
					"page":  1,
					"pages": 1,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{
		Query: "deploy in:#engineering",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "searcher@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data searchMessagesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(data.Matches))
	}
	if data.Matches[0].ChannelID != "C001" {
		t.Errorf("expected channel_id 'C001', got %q", data.Matches[0].ChannelID)
	}
	if data.Matches[0].Text != "Deploying v2.0 now" {
		t.Errorf("expected text 'Deploying v2.0 now', got %q", data.Matches[0].Text)
	}
	if data.Matches[0].Permalink != "https://team.slack.com/archives/C001/p1234567890123456" {
		t.Errorf("unexpected permalink: %q", data.Matches[0].Permalink)
	}
	if data.Total != 2 {
		t.Errorf("expected total 2, got %d", data.Total)
	}
	if data.Pages != 1 {
		t.Errorf("expected pages 1, got %d", data.Pages)
	}
}

func TestSearchMessages_WithoutEmail_Succeeds(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/search.messages" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"messages": map[string]any{
				"matches": []map[string]any{
					{
						"channel": map[string]string{"id": "D_PRIVATE", "name": "dm"},
						"user":    "U002",
						"text":    "private dm",
						"ts":      "2.2",
					},
				},
				"paging": map[string]int{"count": 20, "total": 1, "page": 1, "pages": 1},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{Query: "hello"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var data searchMessagesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(data.Matches) != 1 || data.Matches[0].ChannelID != "D_PRIVATE" {
		t.Fatalf("expected one DM match, got %+v", data.Matches)
	}
}

func TestSearchMessages_MissingQuery(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing query")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchMessages_CountOutOfRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{
		Query: "test",
		Count: 200,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for count out of range")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchMessages_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "missing_scope",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{
		Query: "test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "test@example.com",
	})
	if err == nil {
		t.Fatal("expected error for missing_scope")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestSearchMessages_InvalidSort(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{
		Query: "test",
		Sort:  "invalid_sort",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid sort value")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchMessages_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchMessagesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
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
