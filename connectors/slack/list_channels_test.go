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

	// Pass types explicitly — the new default includes private types which
	// require UserEmail and access-control mocks.
	params, _ := json.Marshal(listChannelsParams{Types: "public_channel"})

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

func TestListChannels_DefaultTypes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]string{"id": "U_CALLER"},
			})
		case "/users.conversations":
			var body usersConversationsRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode users.conversations body: %v", err)
			}
			// public_channel should be stripped — only private types are needed.
			if body.Types != "private_channel,mpim,im" {
				t.Errorf("expected types 'private_channel,mpim,im' for users.conversations, got %q", body.Types)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{"id": "C001"},
					{"id": "D001"},
				},
			})
		case "/conversations.list":
			var body listChannelsRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode: %v", err)
			}
			if body.Types != "public_channel,private_channel,mpim,im" {
				t.Errorf("expected default types 'public_channel,private_channel,mpim,im', got %q", body.Types)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{"id": "C001", "name": "general", "is_private": false, "num_members": 10},
					{"id": "D001", "user": "U_OTHER", "is_private": true, "num_members": 0},
				},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelsAction{conn: conn}

	// No types param — should use the new default (all types).
	params, _ := json.Marshal(listChannelsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
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
		t.Errorf("expected first channel C001, got %q", data.Channels[0].ID)
	}
	if data.Channels[1].ID != "D001" {
		t.Errorf("expected second channel D001, got %q", data.Channels[1].ID)
	}
}

func TestListChannels_IMChannels(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]string{"id": "U_CALLER"},
			})
		case "/users.conversations":
			// Return the IM channel as one the caller belongs to.
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{"id": "D001"},
					{"id": "D002"},
				},
			})
		case "/conversations.list":
			var body listChannelsRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode: %v", err)
			}
			if body.Types != "im" {
				t.Errorf("expected types 'im', got %q", body.Types)
			}
			// IM channels have no name — they have a user field instead.
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{
						"id":         "D001",
						"user":       "U_OTHER",
						"is_private": true,
						"num_members": 0,
					},
					{
						"id":         "D002",
						"user":       "U_ANOTHER",
						"is_private": true,
						"num_members": 0,
					},
					{
						// DM the caller is NOT a member of — should be filtered out.
						"id":         "D999",
						"user":       "U_STRANGER",
						"is_private": true,
						"num_members": 0,
					},
				},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelsAction{conn: conn}

	params, _ := json.Marshal(listChannelsParams{Types: "im"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listChannelsResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	// D999 should be filtered out (not in caller's membership set).
	if len(data.Channels) != 2 {
		t.Fatalf("expected 2 IM channels, got %d", len(data.Channels))
	}
	if data.Channels[0].ID != "D001" {
		t.Errorf("expected first channel D001, got %q", data.Channels[0].ID)
	}
	if data.Channels[0].User != "U_OTHER" {
		t.Errorf("expected user field 'U_OTHER', got %q", data.Channels[0].User)
	}
	if data.Channels[0].Name != "" {
		t.Errorf("expected empty name for IM channel, got %q", data.Channels[0].Name)
	}
	if data.Channels[1].ID != "D002" {
		t.Errorf("expected second channel D002, got %q", data.Channels[1].ID)
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
		Types:  "public_channel",
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

	params, _ := json.Marshal(listChannelsParams{Types: "public_channel"})

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

func TestListChannels_LimitOutOfRange(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listChannelsAction{conn: conn}

	params, _ := json.Marshal(listChannelsParams{Limit: 2000})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for limit out of range")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
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

func TestListChannels_DefaultFallbackWithoutEmail(t *testing.T) {
	t.Parallel()

	// When no types are specified (using the default) and no UserEmail is set,
	// list_channels should gracefully fall back to public_channel only instead
	// of returning a ValidationError. This preserves backward compatibility.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.list" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body listChannelsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if body.Types != "public_channel" {
			t.Errorf("expected fallback types 'public_channel', got %q", body.Types)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channels": []map[string]any{
				{"id": "C001", "name": "general", "is_private": false, "num_members": 5},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelsAction{conn: conn}

	// No types param, no UserEmail — should fall back to public_channel.
	params, _ := json.Marshal(listChannelsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("expected graceful fallback, got error: %v", err)
	}

	var data listChannelsResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(data.Channels) != 1 {
		t.Fatalf("expected 1 channel, got %d", len(data.Channels))
	}
	if data.Channels[0].ID != "C001" {
		t.Errorf("expected channel C001, got %q", data.Channels[0].ID)
	}
}
