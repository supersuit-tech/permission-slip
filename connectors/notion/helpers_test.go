package notion

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validCreds returns a Credentials value with a valid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "ntn_test_token_123",
	})
}

// validOAuthCreds returns a Credentials value with an OAuth access token for tests.
func validOAuthCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "ntn_test_token_123",
	})
}

// notionErrorBody returns a JSON-encoded Notion error response for use in
// httptest handlers.
func notionErrorBody(status int, code, message string) []byte {
	b, _ := json.Marshal(notionErrorResponse{
		Object:  "error",
		Status:  status,
		Code:    code,
		Message: message,
	})
	return b
}

// newTestServer creates an httptest.Server and a NotionConnector pointing at
// it. The caller provides the handler and must call srv.Close() when done.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *NotionConnector) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	conn := newForTest(srv.Client(), srv.URL)
	return srv, conn
}
