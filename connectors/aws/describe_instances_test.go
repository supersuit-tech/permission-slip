package aws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDescribeInstances_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<DescribeInstancesResponse xmlns="http://ec2.amazonaws.com/doc/2016-11-15/">
  <reservationSet>
    <item>
      <instancesSet>
        <item>
          <instanceId>i-1234567890abcdef0</instanceId>
          <instanceType>t2.micro</instanceType>
          <instanceState>
            <code>16</code>
            <name>running</name>
          </instanceState>
          <privateIpAddress>10.0.0.1</privateIpAddress>
          <ipAddress>54.1.2.3</ipAddress>
          <placement>
            <availabilityZone>us-east-1a</availabilityZone>
          </placement>
          <tagSet>
            <item>
              <key>Name</key>
              <value>my-instance</value>
            </item>
          </tagSet>
        </item>
      </instancesSet>
    </item>
  </reservationSet>
</DescribeInstancesResponse>`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	// Override the do method to point at test server by wrapping.
	action := &describeInstancesAction{conn: conn}

	// We need to use the test server URL, so we'll test via the httptest approach.
	// Since do() builds the URL from the host, we test at the action level with a
	// custom server that intercepts requests.
	result, err := executeWithTestServer(t, srv, action, json.RawMessage(`{"region":"us-east-1"}`))
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["count"] != float64(1) {
		t.Errorf("count = %v, want 1", data["count"])
	}
	instances, ok := data["instances"].([]any)
	if !ok || len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %v", data["instances"])
	}
	inst := instances[0].(map[string]any)
	if inst["instance_id"] != "i-1234567890abcdef0" {
		t.Errorf("instance_id = %v, want i-1234567890abcdef0", inst["instance_id"])
	}
	if inst["state"] != "running" {
		t.Errorf("state = %v, want running", inst["state"])
	}
}

func TestDescribeInstances_MissingRegion(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.describe_instances"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.describe_instances",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDescribeInstances_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["aws.describe_instances"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.describe_instances",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

// executeWithTestServer creates a test connector that points at the given test
// server and executes the given action. This bypasses the real AWS hostname
// construction by using a custom connector with the test server's client.
func executeWithTestServer(t *testing.T, srv *httptest.Server, action connectors.Action, params json.RawMessage) (*connectors.ActionResult, error) {
	t.Helper()

	// Create a proxy test server that the action can reach.
	// Since the connector constructs URLs from the host, we intercept at the
	// HTTP transport level.
	proxyConn := &AWSConnector{
		client: srv.Client(),
	}

	// Wrap the action with the proxy connector.
	switch a := action.(type) {
	case *describeInstancesAction:
		a.conn = proxyConn
		// Override do to use test server URL.
		return executeWithProxy(t, srv, proxyConn, a, params)
	default:
		t.Fatalf("unsupported action type for test server: %T", action)
		return nil, nil
	}
}

// executeWithProxy runs an action through a proxy that redirects AWS API calls
// to the test server.
func executeWithProxy(t *testing.T, srv *httptest.Server, conn *AWSConnector, action connectors.Action, params json.RawMessage) (*connectors.ActionResult, error) {
	t.Helper()

	// Create a transport that redirects all requests to the test server.
	transport := srv.Client().Transport
	conn.client = &http.Client{
		Transport: &testTransport{
			inner:   transport,
			testURL: srv.URL,
		},
	}

	return action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "aws.describe_instances",
		Parameters:  params,
		Credentials: validCreds(),
	})
}

// testTransport redirects requests to a test server URL.
type testTransport struct {
	inner   http.RoundTripper
	testURL string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect to test server.
	testReq := req.Clone(req.Context())
	testReq.URL, _ = testReq.URL.Parse(t.testURL + req.URL.Path)
	if req.URL.RawQuery != "" {
		testReq.URL.RawQuery = req.URL.RawQuery
	}
	return t.inner.RoundTrip(testReq)
}
