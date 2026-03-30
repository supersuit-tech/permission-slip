package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestVerifyChannelAccess_PublicChannel_Allowed(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/conversations.info" {
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]any{"id": "C01234567", "is_private": false},
			})
			return
		}
		t.Errorf("unexpected API call: %s", r.URL.Path)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.verifyChannelAccess(t.Context(), validCreds(), "C01234567", "")
	if err != nil {
		t.Fatalf("expected public channel to be allowed without email, got: %v", err)
	}
}

func TestVerifyChannelAccess_PrivateCChannel_RequiresEmail(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/conversations.info" {
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]any{"id": "C01234567", "is_private": true},
			})
			return
		}
		t.Errorf("unexpected API call: %s", r.URL.Path)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.verifyChannelAccess(t.Context(), validCreds(), "C01234567", "")
	if err == nil {
		t.Fatal("expected error for private C-channel without email")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestVerifyChannelAccess_DMChannel_DeniedWithoutEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.verifyChannelAccess(t.Context(), validCreds(), "D01234567", "")
	if err == nil {
		t.Fatal("expected error for DM channel without email")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestVerifyChannelAccess_DMChannel_AllowedWhenMember(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U_ALICE"},
			})
		case "/conversations.members":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"members": []string{"U_ALICE", "U_BOB"},
			})
		default:
			t.Errorf("unexpected API call: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.verifyChannelAccess(t.Context(), validCreds(), "D01234567", "alice@example.com")
	if err != nil {
		t.Fatalf("expected DM access to be allowed for member, got: %v", err)
	}
}

func TestVerifyChannelAccess_DMChannel_DeniedWhenNotMember(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U_CHARLIE"},
			})
		case "/conversations.members":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"members": []string{"U_ALICE", "U_BOB"},
			})
		default:
			t.Errorf("unexpected API call: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.verifyChannelAccess(t.Context(), validCreds(), "D01234567", "charlie@example.com")
	if err == nil {
		t.Fatal("expected error for non-member DM access")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestVerifyChannelAccess_GroupDM_DeniedWhenNotMember(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U_OUTSIDER"},
			})
		case "/conversations.members":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"members": []string{"U_ALICE", "U_BOB", "U_CAROL"},
			})
		default:
			t.Errorf("unexpected API call: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.verifyChannelAccess(t.Context(), validCreds(), "G01234567", "outsider@example.com")
	if err == nil {
		t.Fatal("expected error for non-member group DM access")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestVerifyChannelAccess_NoSlackUserFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/users.lookupByEmail" {
			json.NewEncoder(w).Encode(map[string]any{
				"ok":    false,
				"error": "users_not_found",
			})
			return
		}
		t.Errorf("unexpected API call: %s", r.URL.Path)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.verifyChannelAccess(t.Context(), validCreds(), "D01234567", "nobody@example.com")
	if err == nil {
		t.Fatal("expected error when Slack user not found")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestLookupSlackUserByEmail_EmptyEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	id, err := conn.lookupSlackUserByEmail(t.Context(), validCreds(), "")
	if err != nil {
		t.Fatalf("expected no error for empty email, got: %v", err)
	}
	if id != "" {
		t.Errorf("expected empty user ID for empty email, got %q", id)
	}
}

func TestIsUserInChannel_PaginatedMembers(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if callCount == 1 {
			// First page — target user not present.
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"members": []string{"U001", "U002"},
				"response_metadata": map[string]string{
					"next_cursor": "page2",
				},
			})
		} else {
			// Second page — target user found.
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"members": []string{"U003", "U_TARGET"},
			})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	found, err := conn.isUserInChannel(t.Context(), validCreds(), "C01234567", "U_TARGET")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected user to be found after pagination")
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
}

func TestChannelTypesIncludePrivate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		types string
		want  bool
	}{
		{"public_channel", false},
		{"public_channel,private_channel", true},
		{"im", true},
		{"mpim", true},
		{"public_channel, im", true},
		{"", false},
	}
	for _, tt := range tests {
		got := channelTypesIncludePrivate(tt.types)
		if got != tt.want {
			t.Errorf("channelTypesIncludePrivate(%q) = %v, want %v", tt.types, got, tt.want)
		}
	}
}

func TestReadChannelMessages_DMChannel_RequiresEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(readChannelMessagesParams{
		Channel: "D01234567",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
		// No UserEmail set — should be denied for DM channel.
	})
	if err == nil {
		t.Fatal("expected error for DM channel without email")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestReadChannelMessages_DMChannel_AllowedForMember(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U_ALICE"},
			})
		case "/conversations.members":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"members": []string{"U_ALICE", "U_BOT"},
			})
		case "/conversations.history":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": []map[string]any{
					{"type": "message", "user": "U_ALICE", "text": "hi", "ts": "1.1"},
				},
				"has_more": false,
			})
		default:
			t.Errorf("unexpected API call: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &readChannelMessagesAction{conn: conn}

	params, _ := json.Marshal(readChannelMessagesParams{
		Channel: "D01234567",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.read_channel_messages",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "alice@example.com",
	})
	if err != nil {
		t.Fatalf("expected success for DM member, got: %v", err)
	}
	var data messagesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(data.Messages))
	}
}

func TestSearchMessages_RequiresEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{
		Query: "test query",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
		// No UserEmail — should be denied.
	})
	if err == nil {
		t.Fatal("expected error for search without email")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSearchMessages_FiltersPrivateResults(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users.lookupByEmail":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U_ALICE"},
			})
		case "/search.messages":
			json.NewEncoder(w).Encode(map[string]any{
				"ok": true,
				"messages": map[string]any{
					"matches": []map[string]any{
						{"channel": map[string]string{"id": "C_PUBLIC", "name": "general"}, "user": "U001", "text": "public msg", "ts": "1.1"},
						{"channel": map[string]string{"id": "D_PRIVATE", "name": "dm"}, "user": "U002", "text": "private dm", "ts": "2.2"},
					},
					"paging": map[string]int{"count": 20, "total": 2, "page": 1, "pages": 1},
				},
			})
		case "/conversations.info":
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"channel": map[string]any{"id": "C_PUBLIC", "is_private": false},
			})
		case "/conversations.members":
			// U_ALICE is NOT in D_PRIVATE.
			json.NewEncoder(w).Encode(map[string]any{
				"ok":      true,
				"members": []string{"U_BOB", "U_CAROL"},
			})
		default:
			t.Errorf("unexpected API call: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{
		Query: "test",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "alice@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data searchMessagesResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	// Only the public channel match should remain; the DM should be filtered out.
	if len(data.Matches) != 1 {
		t.Fatalf("expected 1 match after filtering, got %d", len(data.Matches))
	}
	if data.Matches[0].ChannelID != "C_PUBLIC" {
		t.Errorf("expected match from C_PUBLIC, got %s", data.Matches[0].ChannelID)
	}
	if data.Total != 1 {
		t.Errorf("expected total 1 after filtering, got %d", data.Total)
	}
}

func TestFilterPrivateTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"public_channel,private_channel,mpim,im", "private_channel,mpim,im"},
		{"public_channel", ""},
		{"im", "im"},
		{"private_channel,mpim", "private_channel,mpim"},
		{"public_channel, im", "im"},
		{"", ""},
	}
	for _, tt := range tests {
		got := filterPrivateTypes(tt.input)
		if got != tt.want {
			t.Errorf("filterPrivateTypes(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
