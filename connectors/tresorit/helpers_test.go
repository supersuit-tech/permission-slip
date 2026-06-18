package tresorit

import (
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		credKeyAccessKey:   "test-access-key",
		credKeySecretKey:   "test-secret-key",
		credKeyEndpointURL: "http://127.0.0.1:3000",
	})
}

func validCredsWithEndpoint(endpoint string) connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		credKeyAccessKey:   "test-access-key",
		credKeySecretKey:   "test-secret-key",
		credKeyEndpointURL: endpoint,
	})
}

type testTransport struct {
	inner   http.RoundTripper
	testURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	testReq := req.Clone(req.Context())
	parsed, err := url.Parse(t.testURL)
	if err != nil {
		return nil, err
	}
	testReq.URL.Scheme = parsed.Scheme
	testReq.URL.Host = parsed.Host
	return t.inner.RoundTrip(testReq)
}

func newTestConnector(client *http.Client) *TresoritConnector {
	return &TresoritConnector{client: client}
}
