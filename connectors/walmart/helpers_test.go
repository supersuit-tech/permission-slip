package walmart

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validCreds returns a Credentials value with valid consumer_id for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"consumer_id": "test-consumer-id",
		"key_version": "1",
	})
}

// newTestServer creates an httptest.Server that delegates requests to
// the provided handler. If handler is nil, requests return 404.
func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler != nil {
			handler(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

// walmartErrorResponse builds a Walmart error envelope JSON for testing.
func walmartErrorResponse(code int, message string) []byte {
	resp := struct {
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}{
		Errors: []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{{Code: code, Message: message}},
	}
	b, _ := json.Marshal(resp)
	return b
}
