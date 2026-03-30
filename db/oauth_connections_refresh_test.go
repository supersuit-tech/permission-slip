package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestListExpiringOAuthConnections_NoConnections(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	conns, err := db.ListExpiringOAuthConnections(context.Background(), tx, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections, got %d", len(conns))
	}
}

func TestListExpiringOAuthConnections_FindsExpiringSoon(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Connection expiring in 10 minutes (within 15-minute horizon).
	expiryWithin := time.Now().Add(10 * time.Minute)
	refreshVaultID := "00000000-0000-0000-0000-000000000002"
	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_expiring", uid, "google", testhelper.OAuthConnectionOpts{
		RefreshTokenVaultID: &refreshVaultID,
		TokenExpiry:         &expiryWithin,
		Scopes:              []string{"openid"},
	})

	// Connection expiring in 2 hours (outside 15-minute horizon).
	expiryLater := time.Now().Add(2 * time.Hour)
	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_later", uid, "microsoft", testhelper.OAuthConnectionOpts{
		RefreshTokenVaultID: &refreshVaultID,
		TokenExpiry:         &expiryLater,
		Scopes:              []string{"openid"},
	})

	conns, err := db.ListExpiringOAuthConnections(context.Background(), tx, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 expiring connection, got %d", len(conns))
	}
	if conns[0].ID != "oconn_expiring" {
		t.Errorf("expected connection 'oconn_expiring', got %q", conns[0].ID)
	}
}

func TestListExpiringOAuthConnections_SkipsWithoutRefreshToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Connection expiring soon but WITHOUT a refresh token.
	expiryWithin := time.Now().Add(5 * time.Minute)
	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_no_refresh", uid, "google", testhelper.OAuthConnectionOpts{
		TokenExpiry: &expiryWithin,
		Scopes:      []string{"openid"},
	})

	conns, err := db.ListExpiringOAuthConnections(context.Background(), tx, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections (no refresh token), got %d", len(conns))
	}
}

func TestListExpiringOAuthConnections_SkipsNonActive(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	expiryWithin := time.Now().Add(5 * time.Minute)
	refreshVaultID := "00000000-0000-0000-0000-000000000002"
	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_needs_reauth", uid, "google", testhelper.OAuthConnectionOpts{
		RefreshTokenVaultID: &refreshVaultID,
		TokenExpiry:         &expiryWithin,
		Status:              "needs_reauth",
		Scopes:              []string{"openid"},
	})

	conns, err := db.ListExpiringOAuthConnections(context.Background(), tx, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conns) != 0 {
		t.Fatalf("expected 0 connections (non-active status), got %d", len(conns))
	}
}

func TestGetRequiredCredentialByActionType_OAuth(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.InsertConnector(t, tx, "google")
	testhelper.InsertConnectorAction(t, tx, "google", "google.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, "google", "google", "google", []string{"https://www.googleapis.com/auth/gmail.send"})

	rc, err := db.GetRequiredCredentialByActionType(context.Background(), tx, "google.send_email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc == nil {
		t.Fatal("expected non-nil required credential")
	}
	if rc.AuthType != "oauth2" {
		t.Errorf("expected auth_type 'oauth2', got %q", rc.AuthType)
	}
	if rc.OAuthProvider == nil || *rc.OAuthProvider != "google" {
		t.Errorf("expected oauth_provider 'google', got %v", rc.OAuthProvider)
	}
	if len(rc.OAuthScopes) != 1 || rc.OAuthScopes[0] != "https://www.googleapis.com/auth/gmail.send" {
		t.Errorf("unexpected scopes: %v", rc.OAuthScopes)
	}
}

// ── GetRequiredCredentialsByActionType (multi-credential) ────────────────────

func TestGetRequiredCredentialsByActionType_MultipleAuthMethods(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.InsertConnector(t, tx, "trello")
	testhelper.InsertConnectorAction(t, tx, "trello", "trello.create_card", "Create Card")
	// Insert two non-oauth2 credentials for the same connector (different services).
	// Mixed oauth2/non-oauth2 types are now prevented at the schema level.
	testhelper.InsertConnectorRequiredCredential(t, tx, "trello", "trello_api", "api_key")
	testhelper.InsertConnectorRequiredCredential(t, tx, "trello", "trello_token", "basic")

	creds, err := db.GetRequiredCredentialsByActionType(context.Background(), tx, "trello.create_card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(creds))
	}
	// Both are non-oauth2, ordered by service name.
	if creds[0].AuthType != "api_key" {
		t.Errorf("expected first credential to be api_key, got %q", creds[0].AuthType)
	}
	if creds[0].Service != "trello_api" {
		t.Errorf("expected first credential service 'trello_api', got %q", creds[0].Service)
	}
	if creds[1].AuthType != "basic" {
		t.Errorf("expected second credential to be basic, got %q", creds[1].AuthType)
	}
	if creds[1].Service != "trello_token" {
		t.Errorf("expected second credential service 'trello_token', got %q", creds[1].Service)
	}
}

func TestGetRequiredCredentialsByActionType_SingleOAuth(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	testhelper.InsertConnector(t, tx, "msft")
	testhelper.InsertConnectorAction(t, tx, "msft", "msft.send_email", "Send")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, "msft", "msft", "microsoft", []string{"Mail.Send"})

	creds, err := db.GetRequiredCredentialsByActionType(context.Background(), tx, "msft.send_email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(creds))
	}
	if creds[0].AuthType != "oauth2" {
		t.Errorf("expected oauth2, got %q", creds[0].AuthType)
	}
}

func TestGetRequiredCredentialsByActionType_NoResults(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	creds, err := db.GetRequiredCredentialsByActionType(context.Background(), tx, "nonexistent.action")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected 0 credentials for nonexistent action, got %d", len(creds))
	}
}
