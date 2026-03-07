package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validCreds returns a Credentials value with a valid bot token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"bot_token": "test-bot-token-123",
	})
}

// mockServer creates an httptest server that verifies the expected HTTP method
// and path, then responds with the given status code and JSON body. It returns
// a DiscordConnector wired to the mock server.
//
// For 204 No Content responses, pass nil for responseBody.
func mockServer(t *testing.T, expectedMethod, expectedPath string, statusCode int, responseBody any) (*DiscordConnector, func()) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != expectedMethod {
			t.Errorf("expected %s, got %s", expectedMethod, r.Method)
		}
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bot test-bot-token-123" {
			t.Errorf("expected Bot token in Authorization header, got %q", got)
		}

		if statusCode == http.StatusNoContent || responseBody == nil {
			w.WriteHeader(statusCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(responseBody)
	}))

	conn := newForTest(srv.Client(), srv.URL)
	return conn, srv.Close
}
