package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListChannels_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/conversations.list" {
			t.Errorf("expected path /conversations.list, got %s", r.URL.Path)
		}

		var body listChannelsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Types != "public_channel" {
			t.Errorf("expected types 'public_channel', got %q", body.Types)
		}
		if body.Limit != 100 {
			t.Errorf("expected limit 100, got %d", body.Limit)
		}
		if !body.ExcludeArchived {
			t.Error("expected exclude_archived to be true")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channels": []map[string]any{
				{
					"id":          "C001",
					"name":        "general",
					"is_private":  false,
					"is_archived": false,
					"num_members": 42,
					"topic":       map[string]string{"value": "General discussion"},
					"purpose":     map[string]string{"value": "Company-wide announcements"},
				},
				{
					"id":          "C002",
					"name":        "engineering",
					"is_private":  false,
					"is_archived": false,
					"num_members": 15,
					"topic":       map[string]string{"value": ""},
					"purpose":     map[string]string{"value": "Engineering team"},
				},
			},
			"response_metadata": map[string]string{
				"next_cursor": "dGVhbTpDMDI=",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelsAction{conn: conn}

	params, _ := json.Marshal(listChannelsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listChannelsResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(data.Channels))
	}
	if data.Channels[0].ID != "C001" {
		t.Errorf("expected first channel ID 'C001', got %q", data.Channels[0].ID)
	}
	if data.Channels[0].Name != "general" {
		t.Errorf("expected first channel name 'general', got %q", data.Channels[0].Name)
	}
	if data.Channels[0].NumMembers != 42 {
		t.Errorf("expected 42 members, got %d", data.Channels[0].NumMembers)
	}
	if data.Channels[0].Topic != "General discussion" {
		t.Errorf("expected topic 'General discussion', got %q", data.Channels[0].Topic)
	}
	if data.NextCursor != "dGVhbTpDMDI=" {
		t.Errorf("expected next_cursor 'dGVhbTpDMDI=', got %q", data.NextCursor)
	}
}

func TestListChannels_WithPagination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body listChannelsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Cursor != "dGVhbTpDMDI=" {
			t.Errorf("expected cursor 'dGVhbTpDMDI=', got %q", body.Cursor)
		}
		if body.Limit != 10 {
			t.Errorf("expected limit 10, got %d", body.Limit)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":       true,
			"channels": []map[string]any{},
			"response_metadata": map[string]string{
				"next_cursor": "",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelsAction{conn: conn}

	params, _ := json.Marshal(listChannelsParams{
		Limit:  10,
		Cursor: "dGVhbTpDMDI=",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listChannelsResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Channels) != 0 {
		t.Errorf("expected 0 channels, got %d", len(data.Channels))
	}
}

func TestListChannels_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelsAction{conn: conn}

	params, _ := json.Marshal(listChannelsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
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

func TestListChannels_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listChannelsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
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
