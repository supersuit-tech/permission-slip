package linear

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// validCreds returns a Credentials value with a valid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "lin_api_test_key_123",
	})
}

// graphQLHandler is a test helper that validates incoming GraphQL requests
// and returns a canned response. It checks that:
//   - The request method is POST
//   - Content-Type is application/json
//   - Authorization header is present
//   - The body is a valid GraphQL request with a query field
type graphQLHandler struct {
	t        *testing.T
	response any    // will be JSON-marshaled and returned
	status   int    // HTTP status code (default 200)
	wantAuth string // expected Authorization header value (optional)
}

func (h *graphQLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.t.Helper()

	// Validate HTTP method.
	if r.Method != http.MethodPost {
		h.t.Errorf("expected POST, got %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Validate Content-Type.
	ct := r.Header.Get("Content-Type")
	if ct != "application/json" {
		h.t.Errorf("expected Content-Type application/json, got %q", ct)
		w.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	// Validate Authorization header is present.
	auth := r.Header.Get("Authorization")
	if auth == "" {
		h.t.Errorf("missing Authorization header")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if h.wantAuth != "" && auth != h.wantAuth {
		h.t.Errorf("Authorization = %q, want %q", auth, h.wantAuth)
	}

	// Validate body is a valid GraphQL request.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.t.Fatalf("reading request body: %v", err)
	}

	var gqlReq struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables,omitempty"`
	}
	if err := json.Unmarshal(body, &gqlReq); err != nil {
		h.t.Errorf("request body is not valid JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if gqlReq.Query == "" {
		h.t.Errorf("GraphQL query field is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Return the canned response.
	status := h.status
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	respBytes, err := json.Marshal(h.response)
	if err != nil {
		h.t.Fatalf("marshaling test response: %v", err)
	}
	_, _ = w.Write(respBytes)
}

// newTestServer creates an httptest server with the graphQLHandler and returns
// a LinearConnector pointing at it. The server is automatically cleaned up
// when the test finishes.
func newTestServer(t *testing.T, handler *graphQLHandler) (*LinearConnector, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	conn := newForTest(server.Client(), server.URL)
	return conn, server
}
