package tresorit

import (
	"context"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// testListBuckets may be replaced in tests to avoid a running gateway.
var testListBuckets = func(ctx context.Context, conn *TresoritConnector, creds connectors.Credentials, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	_, err := conn.do(ctx, creds, "GET", "/", "", nil, "")
	return err
}

// TestGatewayConnection verifies reachability of the local Tresorit S3 gateway
// by issuing a ListBuckets request.
func TestGatewayConnection(ctx context.Context, conn *TresoritConnector, creds connectors.Credentials, timeout time.Duration) error {
	if err := validateCredentialShape(creds); err != nil {
		return err
	}
	if err := testListBuckets(ctx, conn, creds, timeout); err != nil {
		return mapGatewayError(err)
	}
	return nil
}

func mapGatewayError(err error) error {
	if err == nil {
		return nil
	}
	if connectors.IsValidationError(err) || connectors.IsAuthError(err) ||
		connectors.IsExternalError(err) || connectors.IsTimeoutError(err) {
		return err
	}
	return &connectors.ExternalError{
		Message: "Tresorit gateway unreachable — confirm the Docker container is running and endpoint_url points to it (default http://127.0.0.1:3000)",
	}
}
