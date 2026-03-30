package amadeus

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// validCreds returns a Credentials value with valid client_id and
// client_secret for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"client_id":     "test-client-id",
		"client_secret": "test-client-secret",
	})
}

// tokenJSON is the standard token response returned by the mock token endpoint.
var tokenJSON = `{
	"type": "amadeusOAuth2Token",
	"username": "test@example.com",
	"application_name": "Test App",
	"client_id": "test-client-id",
	"token_type": "Bearer",
	"access_token": "test-access-token-123",
	"expires_in": 1799,
	"state": "approved",
	"scope": ""
}`

// newTestServer creates an httptest.Server that handles the token endpoint
// and delegates all other requests to the provided handler. If handler is
// nil, non-token requests return 404.
func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Token endpoint.
		if r.URL.Path == "/v1/security/oauth2/token" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(tokenJSON))
			return
		}

		if handler != nil {
			handler(w, r)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
}

// newTestServerWithTokenError creates a test server where the token endpoint
// returns the specified status code and body.
func newTestServerWithTokenError(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/security/oauth2/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			_, _ = w.Write([]byte(body))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

// amadeusErrorResponse builds an Amadeus error envelope JSON for testing.
func amadeusErrorResponse(status, code int, title, detail string) []byte {
	resp := struct {
		Errors []amadeusAPIError `json:"errors"`
	}{
		Errors: []amadeusAPIError{{
			Status: status,
			Code:   code,
			Title:  title,
			Detail: detail,
		}},
	}
	b, _ := json.Marshal(resp)
	return b
}
