package figma

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// validCreds returns a Credentials value with a valid personal access token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"personal_access_token": "figd_test_token_123",
	})
}

// figmaErrorBody returns a JSON-encoded Figma error response for use in
// httptest handlers.
func figmaErrorBody(status int, errMsg string) []byte {
	b, _ := json.Marshal(figmaErrorResponse{
		Status: status,
		Err:    errMsg,
	})
	return b
}

// newTestServer creates an httptest.Server and a FigmaConnector pointing at
// it. The caller provides the handler; cleanup is automatic via t.Cleanup.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *FigmaConnector) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	conn := newForTest(srv.Client(), srv.URL)
	return srv, conn
}
