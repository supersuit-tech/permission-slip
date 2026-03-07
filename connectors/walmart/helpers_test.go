package walmart

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

var (
	testKeyOnce sync.Once
	testKeyPEM  string
)

// testPrivateKeyPEM returns a PEM-encoded RSA private key for testing.
// The key is generated once and reused across tests.
func testPrivateKeyPEM() string {
	testKeyOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic("generating test RSA key: " + err.Error())
		}
		der := x509.MarshalPKCS1PrivateKey(key)
		testKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: der,
		}))
	})
	return testKeyPEM
}

// validCreds returns a Credentials value with valid consumer_id and
// private_key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"consumer_id": "test-consumer-id",
		"key_version": "1",
		"private_key": testPrivateKeyPEM(),
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
