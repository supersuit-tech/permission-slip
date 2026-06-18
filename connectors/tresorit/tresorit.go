// Package tresorit implements the Tresorit connector for Permission Slip.
// It speaks the S3-compatible REST API exposed by Tresorit's local gateway
// Docker container. All encryption happens inside the gateway — this connector
// is a SigV4-signed S3 client against a custom loopback endpoint.
package tresorit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/connectors/s3sigv4"
)

const (
	defaultTimeout = 30 * time.Second
	fixedRegion    = "us-east-1"
	fixedService   = "s3"

	credKeyAccessKey   = "access_key"
	credKeySecretKey   = "secret_key"
	credKeyEndpointURL = "endpoint_url"

	// maxResponseBytes caps upstream response size.
	maxResponseBytes = 10 * 1024 * 1024 // 10 MB
	// maxUploadBytes caps upload payload size.
	maxUploadBytes = 150 << 20 // 150 MB
)

// TresoritConnector owns the shared HTTP client used by all Tresorit actions.
type TresoritConnector struct {
	client *http.Client
}

// New creates a TresoritConnector with sensible defaults.
func New() *TresoritConnector {
	return &TresoritConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a TresoritConnector that uses a custom HTTP client.
func newForTest(client *http.Client) *TresoritConnector {
	return &TresoritConnector{client: client}
}

// ID returns "tresorit".
func (c *TresoritConnector) ID() string { return "tresorit" }

// Actions returns the registered action handlers keyed by action_type.
func (c *TresoritConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"tresorit.list_files":    &listFilesAction{conn: c},
		"tresorit.download_file": &downloadFileAction{conn: c},
		"tresorit.upload_file":   &uploadFileAction{conn: c},
		"tresorit.create_folder": &createFolderAction{conn: c},
		"tresorit.delete_file":   &deleteFileAction{conn: c},
	}
}

// ValidateCredentials checks credential shape and verifies the local gateway
// is reachable with a live ListBuckets call.
func (c *TresoritConnector) ValidateCredentials(ctx context.Context, creds connectors.Credentials) error {
	if err := validateCredentialShape(creds); err != nil {
		return err
	}

	timeout := defaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	return TestGatewayConnection(ctx, c, creds, timeout)
}

func validateCredentialShape(creds connectors.Credentials) error {
	accessKey, ok := creds.Get(credKeyAccessKey)
	if !ok || strings.TrimSpace(accessKey) == "" {
		return &connectors.ValidationError{Message: "missing required credential: access_key"}
	}
	secretKey, ok := creds.Get(credKeySecretKey)
	if !ok || strings.TrimSpace(secretKey) == "" {
		return &connectors.ValidationError{Message: "missing required credential: secret_key"}
	}
	endpoint, ok := creds.Get(credKeyEndpointURL)
	if !ok || strings.TrimSpace(endpoint) == "" {
		return &connectors.ValidationError{Message: "missing required credential: endpoint_url"}
	}
	if _, err := parseEndpoint(endpoint); err != nil {
		return err
	}
	return nil
}

func parseEndpoint(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	u, err := url.Parse(raw)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid endpoint_url: %v", err)}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, &connectors.ValidationError{Message: "endpoint_url must use http or https"}
	}
	if u.Host == "" {
		return nil, &connectors.ValidationError{Message: "endpoint_url must include a host (e.g. http://127.0.0.1:3000)"}
	}
	return u, nil
}

func endpointURL(creds connectors.Credentials) (string, error) {
	raw, _ := creds.Get(credKeyEndpointURL)
	u, err := parseEndpoint(raw)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(u.String(), "/"), nil
}

// do sends a SigV4-signed S3 request to the Tresorit gateway.
func (c *TresoritConnector) do(ctx context.Context, creds connectors.Credentials, method, path, query string, body []byte, contentType string) ([]byte, error) {
	base, err := endpointURL(creds)
	if err != nil {
		return nil, err
	}

	fullURL := base + path
	if query != "" {
		fullURL += "?" + query
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("creating request: %v", err)}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	u, _ := parseEndpoint(base)
	req.Host = u.Host

	accessKey, _ := creds.Get(credKeyAccessKey)
	secretKey, _ := creds.Get(credKeySecretKey)
	if err := s3sigv4.SignRequest(req, s3sigv4.Credentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
	}, body, s3sigv4.SigningConfig{
		Region:  fixedRegion,
		Service: fixedService,
	}); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("signing request: %v", err)}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("Tresorit gateway request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, &connectors.CanceledError{Message: "Tresorit gateway request canceled"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("Tresorit gateway request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, respBytes); err != nil {
		return nil, err
	}

	return respBytes, nil
}

func objectPath(tresor, key string) string {
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "/" + s3sigv4.URIEncodePath(tresor)
	}
	return "/" + s3sigv4.URIEncodePath(tresor+"/"+key)
}

func ptrBool(b bool) *bool { return &b }
