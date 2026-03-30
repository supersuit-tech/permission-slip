// Package doordash implements the DoorDash Drive connector for the Permission
// Slip connector execution layer. DoorDash Drive is a delivery-as-a-service API
// that moves items from point A to point B — it is NOT consumer food ordering.
//
// Auth: self-signed JWT tokens (HS256). The connector generates a short-lived
// JWT (5 min) on each request using three credentials:
//   - developer_id: identifies the developer account (JWT "iss" claim)
//   - key_id: identifies the specific API key (JWT "kid" header)
//   - signing_secret: base64url-encoded HMAC key (decoded before signing)
//
// The signing_secret is base64url-encoded as provided by the DoorDash Developer
// Portal. The connector decodes it before use — passing the raw string would
// produce invalid signatures. See [DoorDash JWT docs] for details.
//
// [DoorDash JWT docs]: https://developer.doordash.com/en-US/docs/drive/reference/JWTs/
package doordash

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	productionBaseURL = "https://openapi.doordash.com"
	defaultTimeout    = 30 * time.Second

	// JWT token lifetime — DoorDash recommends short-lived tokens.
	jwtLifetime = 5 * time.Minute

	credKeyDeveloperID   = "developer_id"
	credKeyKeyID         = "key_id"
	credKeySigningSecret = "signing_secret"
)

// deliveryStatuses is the canonical list of DoorDash delivery statuses.
// Used by both list_deliveries validation and the manifest JSON schema so
// they can never drift apart.
var deliveryStatuses = []string{
	"created",
	"confirmed",
	"enroute_to_pickup",
	"arrived_at_pickup",
	"picked_up",
	"enroute_to_dropoff",
	"arrived_at_dropoff",
	"delivered",
	"cancelled",
	"enroute_to_return",
	"returned",
}

// DoorDashConnector owns the shared HTTP client and base URL used by all
// DoorDash Drive actions.
type DoorDashConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a DoorDashConnector with sensible defaults.
func New() *DoorDashConnector {
	return &DoorDashConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: productionBaseURL,
	}
}

// newForTest creates a DoorDashConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *DoorDashConnector {
	return &DoorDashConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "doordash", matching the connectors.id in the database.
func (c *DoorDashConnector) ID() string { return "doordash" }

// Actions returns the registered action handlers keyed by action_type.
func (c *DoorDashConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"doordash.get_quote":        &getQuoteAction{conn: c},
		"doordash.create_delivery":  &createDeliveryAction{conn: c},
		"doordash.get_delivery":     &getDeliveryAction{conn: c},
		"doordash.cancel_delivery":  &cancelDeliveryAction{conn: c},
		"doordash.list_deliveries":  &listDeliveriesAction{conn: c},
	}
}

// ValidateCredentials checks that all three DoorDash Drive credentials are
// present: developer_id, key_id, and signing_secret.
func (c *DoorDashConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	for _, key := range []string{credKeyDeveloperID, credKeyKeyID, credKeySigningSecret} {
		v, ok := creds.Get(key)
		if !ok || v == "" {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("missing required credential: %s — find your credentials at https://developer.doordash.com/portal/integration/drive", key),
			}
		}
	}
	return nil
}

// generateJWT creates a short-lived JWT token for DoorDash API authentication.
// The token uses HS256 signing with a custom header field "dd-ver" set to "DD-JWT-V1".
func generateJWT(creds connectors.Credentials) (string, error) {
	developerID, _ := creds.Get(credKeyDeveloperID)
	keyID, _ := creds.Get(credKeyKeyID)
	signingSecret, _ := creds.Get(credKeySigningSecret)

	if developerID == "" || keyID == "" || signingSecret == "" {
		return "", &connectors.ValidationError{Message: "DoorDash credentials (developer_id, key_id, signing_secret) are required"}
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"aud": "doordash",
		"iss": developerID,
		"exp": jwt.NewNumericDate(now.Add(jwtLifetime)),
		"iat": jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["dd-ver"] = "DD-JWT-V1"
	token.Header["kid"] = keyID

	// DoorDash provides the signing secret as base64-encoded bytes.
	// Decode before using as the HMAC-SHA256 key.
	decodedSecret, err := base64.RawURLEncoding.DecodeString(signingSecret)
	if err != nil {
		return "", &connectors.AuthError{
			Message: "signing_secret is not valid base64url — verify your credentials at https://developer.doordash.com/portal/integration/drive",
		}
	}

	signed, signErr := token.SignedString(decodedSecret)
	if signErr != nil {
		return "", &connectors.AuthError{Message: "failed to sign JWT — verify that your signing_secret is valid"}
	}
	return signed, nil
}

// newUUID generates a UUID v4 string for use as external_delivery_id.
func newUUID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}

// do is the shared request lifecycle for all DoorDash actions. It generates a
// fresh JWT, marshals reqBody as JSON, sends the request, checks the response
// status, and unmarshals the response into respBody.
func (c *DoorDashConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	token, err := generateJWT(creds)
	if err != nil {
		return err
	}

	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("marshaling request body: %v", err)}
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("creating request: %v", err)}
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("DoorDash API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "DoorDash API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("DoorDash API request failed: %v", err)}
	}
	defer resp.Body.Close()

	const maxResponseSize = 5 << 20
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing DoorDash response: %v", err)}
		}
	}
	return nil
}
