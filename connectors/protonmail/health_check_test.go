package protonmail

import (
	"errors"
	"syscall"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestTestBridgeConnection_success(t *testing.T) {
	oldIMAP := testIMAPConn
	oldSMTP := testSMTPConn
	testIMAPConn = func(_ connectors.Credentials, _ time.Duration) error { return nil }
	testSMTPConn = func(_ connectors.Credentials, _ time.Duration) error { return nil }
	t.Cleanup(func() {
		testIMAPConn = oldIMAP
		testSMTPConn = oldSMTP
	})

	if err := TestBridgeConnection(validCreds(), time.Second); err != nil {
		t.Fatalf("TestBridgeConnection() error = %v, want nil", err)
	}
}

func TestTestBridgeConnection_imapFailure(t *testing.T) {
	oldIMAP := testIMAPConn
	oldSMTP := testSMTPConn
	testIMAPConn = func(_ connectors.Credentials, _ time.Duration) error {
		return &connectors.ExternalError{Message: "dial tcp 127.0.0.1:1143: connect: connection refused"}
	}
	testSMTPConn = func(_ connectors.Credentials, _ time.Duration) error { return nil }
	t.Cleanup(func() {
		testIMAPConn = oldIMAP
		testSMTPConn = oldSMTP
	})

	err := TestBridgeConnection(validCreds(), time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsExternalError(err) {
		t.Fatalf("expected ExternalError, got %T: %v", err, err)
	}
	var ee *connectors.ExternalError
	if !errors.As(err, &ee) {
		t.Fatal("expected ExternalError")
	}
	if ee.Message == "" || ee.Message == "dial tcp 127.0.0.1:1143: connect: connection refused" {
		t.Errorf("expected actionable message, got %q", ee.Message)
	}
}

func TestTestBridgeConnection_authFailure(t *testing.T) {
	oldIMAP := testIMAPConn
	testIMAPConn = func(_ connectors.Credentials, _ time.Duration) error {
		return &connectors.AuthError{Message: "IMAP login failed: invalid credentials"}
	}
	t.Cleanup(func() { testIMAPConn = oldIMAP })

	err := TestBridgeConnection(validCreds(), time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsAuthError(err) {
		t.Fatalf("expected AuthError, got %T: %v", err, err)
	}
	var ae *connectors.AuthError
	if !errors.As(err, &ae) {
		t.Fatal("expected AuthError")
	}
	if ae.Message == "" {
		t.Error("expected non-empty auth message")
	}
}

func TestTestBridgeConnection_timeout(t *testing.T) {
	oldIMAP := testIMAPConn
	testIMAPConn = func(_ connectors.Credentials, _ time.Duration) error {
		return &connectors.TimeoutError{Message: "IMAP connection timed out"}
	}
	t.Cleanup(func() { testIMAPConn = oldIMAP })

	err := TestBridgeConnection(validCreds(), time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsTimeoutError(err) {
		t.Fatalf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestIsConnectionRefused(t *testing.T) {
	t.Parallel()
	if !isConnectionRefused(syscall.ECONNREFUSED) {
		t.Error("expected ECONNREFUSED to match")
	}
	if !isConnectionRefused(errors.New("dial tcp 127.0.0.1:1143: connect: connection refused")) {
		t.Error("expected connection refused message to match")
	}
	if isConnectionRefused(errors.New("no such host")) {
		t.Error("expected unrelated error not to match")
	}
}

func TestProtonMailConnector_ValidateCredentials_usesBridgeTest(t *testing.T) {
	oldIMAP := testIMAPConn
	oldSMTP := testSMTPConn
	called := false
	testIMAPConn = func(_ connectors.Credentials, _ time.Duration) error {
		called = true
		return nil
	}
	testSMTPConn = func(_ connectors.Credentials, _ time.Duration) error { return nil }
	t.Cleanup(func() {
		testIMAPConn = oldIMAP
		testSMTPConn = oldSMTP
	})

	c := New()
	if err := c.ValidateCredentials(t.Context(), validCreds()); err != nil {
		t.Fatalf("ValidateCredentials() error = %v", err)
	}
	if !called {
		t.Error("expected IMAP health check to run")
	}
}
