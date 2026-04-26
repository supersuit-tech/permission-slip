package aws

import (
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// validCreds returns a Credentials value with valid AWS credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_key_id":     "AKIAIOSFODNN7EXAMPLE",
		"secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"region":            "us-east-1",
	})
}

// testTransport redirects requests to a test server URL.
// Used across all action tests to intercept AWS API calls.
type testTransport struct {
	inner   http.RoundTripper
	testURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	testReq := req.Clone(req.Context())
	testReq.URL, _ = testReq.URL.Parse(t.testURL + req.URL.Path)
	if req.URL.RawQuery != "" {
		testReq.URL.RawQuery = req.URL.RawQuery
	}
	return t.inner.RoundTrip(testReq)
}
