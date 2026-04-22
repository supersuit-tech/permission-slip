package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchMessages_UsesAccessToken(t *testing.T) {
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
	creds := connectors.NewCredentials(map[string]string{
		"access_token": "xoxp-user-token",
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

	if receivedToken != "Bearer xoxp-user-token" {
		t.Errorf("expected access token in Authorization header, got %q", receivedToken)
	}
}

func TestOAuthScopes_IncludesRequiredUserScopes(t *testing.T) {
	t.Parallel()

	want := map[string]struct{}{
		"channels:history":     {},
		"channels:read":        {},
		"channels:write":       {},
		"channels:write.topic": {},
		"chat:write":           {},
		"files:read":           {},
		"files:write":          {},
		"groups:history":       {},
		"groups:read":          {},
		"groups:write":         {},
		"im:history":           {},
		"im:read":              {},
		"im:write":             {},
		"mpim:history":         {},
		"mpim:read":            {},
		"mpim:write":           {},
		"reactions:read":       {},
		"reactions:write":      {},
		"search:read":          {},
		"users:read":           {},
		"users:read.email":     {},
	}
	for _, s := range OAuthScopes {
		delete(want, s)
	}
	for s := range want {
		t.Errorf("missing OAuth scope: %q", s)
	}
}
