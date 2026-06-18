package tresorit

import (
	"context"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestTestGatewayConnection_success(t *testing.T) {
	old := testListBuckets
	testListBuckets = func(_ context.Context, _ *TresoritConnector, _ connectors.Credentials, _ time.Duration) error {
		return nil
	}
	t.Cleanup(func() { testListBuckets = old })

	if err := TestGatewayConnection(t.Context(), New(), validCreds(), time.Second); err != nil {
		t.Fatalf("TestGatewayConnection() error = %v, want nil", err)
	}
}

func TestTestGatewayConnection_authFailure(t *testing.T) {
	old := testListBuckets
	testListBuckets = func(_ context.Context, _ *TresoritConnector, _ connectors.Credentials, _ time.Duration) error {
		return &connectors.AuthError{Message: "SignatureDoesNotMatch"}
	}
	t.Cleanup(func() { testListBuckets = old })

	err := TestGatewayConnection(t.Context(), New(), validCreds(), time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsAuthError(err) {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}
}
