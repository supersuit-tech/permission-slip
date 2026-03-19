package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchMessages_PrefersUserToken(t *testing.T) {
	t.Parallel()

	// Track which token was used in the search request.
	var receivedToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/users.lookupByEmail" {
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U_TEST"},
			})
			return
		}
		if r.URL.Path == "/search.messages" {
			receivedToken = r.Header.Get("Authorization")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"messages": map[string]any{
				"matches": []any{},
				"paging":  map[string]int{"count": 20, "total": 0, "page": 1, "pages": 0},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{Query: "test"})

	// Provide both bot token (access_token) and user token.
	creds := connectors.NewCredentials(map[string]string{
		"access_token":      "xoxb-bot-token",
		"user_access_token": "xoxp-user-token",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: creds,
		UserEmail:   "test@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The user token should have been used, not the bot token.
	if receivedToken != "Bearer xoxp-user-token" {
		t.Errorf("expected user token in Authorization header, got %q", receivedToken)
	}
}

func TestSearchMessages_FallsBackToBotToken(t *testing.T) {
	t.Parallel()

	var receivedToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/users.lookupByEmail" {
			json.NewEncoder(w).Encode(map[string]any{
				"ok":   true,
				"user": map[string]any{"id": "U_TEST"},
			})
			return
		}
		if r.URL.Path == "/search.messages" {
			receivedToken = r.Header.Get("Authorization")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"messages": map[string]any{
				"matches": []any{},
				"paging":  map[string]int{"count": 20, "total": 0, "page": 1, "pages": 0},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &searchMessagesAction{conn: conn}

	params, _ := json.Marshal(searchMessagesParams{Query: "test"})

	// Only bot token, no user token.
	creds := connectors.NewCredentials(map[string]string{
		"access_token": "xoxb-bot-token",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.search_messages",
		Parameters:  params,
		Credentials: creds,
		UserEmail:   "test@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedToken != "Bearer xoxb-bot-token" {
		t.Errorf("expected bot token in Authorization header, got %q", receivedToken)
	}
}

func TestOAuthUserScopes_AreSearchScopes(t *testing.T) {
	t.Parallel()

	// Verify OAuthUserScopes contains all search:read.* scopes.
	expected := map[string]bool{
		"search:read.public":  true,
		"search:read.private": true,
		"search:read.im":      true,
		"search:read.mpim":    true,
		"search:read.files":   true,
	}
	for _, s := range OAuthUserScopes {
		if !expected[s] {
			t.Errorf("unexpected user scope: %q", s)
		}
		delete(expected, s)
	}
	for s := range expected {
		t.Errorf("missing user scope: %q", s)
	}
}

func TestOAuthScopes_DoNotContainSearchScopes(t *testing.T) {
	t.Parallel()

	// Verify search scopes have been removed from bot scopes.
	for _, s := range OAuthScopes {
		if len(s) >= 11 && s[:11] == "search:read" {
			t.Errorf("bot OAuthScopes should not contain search scope %q (moved to OAuthUserScopes)", s)
		}
	}
}
