package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestReloadOAuthConnectionIfConcurrentRefreshSucceeded_DetectsNewerExpiry(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "oconn_reload"
	refreshVault := "00000000-0000-0000-0000-000000000002"
	expiry := time.Now().Add(10 * time.Minute)
	testhelper.InsertOAuthConnectionFull(t, tx, connID, uid, "google", testhelper.OAuthConnectionOpts{
		RefreshTokenVaultID: &refreshVault,
		TokenExpiry:         &expiry,
		Scopes:              []string{"openid"},
	})

	before, err := db.GetOAuthConnectionByID(context.Background(), tx, connID)
	if err != nil || before == nil {
		t.Fatalf("get connection: %v", before)
	}

	newExpiry := time.Now().Add(2 * time.Hour)
	testhelper.MustExec(t, tx,
		`UPDATE oauth_connections SET token_expiry = $1, updated_at = now() WHERE id = $2`,
		newExpiry, connID,
	)

	fresh, skip, err := db.ReloadOAuthConnectionIfConcurrentRefreshSucceeded(context.Background(), tx, connID, uid, before.AccessTokenVaultID, before.TokenExpiry)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !skip || fresh == nil {
		t.Fatalf("expected skip=true with fresh row, got skip=%v fresh=%v", skip, fresh)
	}
	if fresh.TokenExpiry == nil || fresh.TokenExpiry.Sub(newExpiry).Abs() > time.Second {
		t.Errorf("expected token_expiry ~%v, got %v", newExpiry, fresh.TokenExpiry)
	}
}

func TestGetOAuthConnectionByProvider_ReturnsMostRecent(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	refreshVault := "00000000-0000-0000-0000-000000000002"
	expiry := time.Now().Add(time.Hour)

	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_old", uid, "slack", testhelper.OAuthConnectionOpts{
		RefreshTokenVaultID: &refreshVault,
		TokenExpiry:         &expiry,
		Scopes:              []string{"channels:read"},
	})
	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_new", uid, "slack", testhelper.OAuthConnectionOpts{
		RefreshTokenVaultID: &refreshVault,
		TokenExpiry:         &expiry,
		Scopes:              []string{"channels:write"},
	})
	testhelper.MustExec(t, tx,
		`UPDATE oauth_connections SET created_at = now() - interval '2 days' WHERE id = $1`,
		"oconn_old",
	)

	conn, err := db.GetOAuthConnectionByProvider(context.Background(), tx, uid, "slack")
	if err != nil {
		t.Fatalf("GetOAuthConnectionByProvider: %v", err)
	}
	if conn == nil || conn.ID != "oconn_new" {
		t.Fatalf("expected newest connection oconn_new, got %+v", conn)
	}
}

func TestReloadOAuthConnectionIfConcurrentRefreshSucceeded_NeedsReauthDoesNotSkip(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "oconn_needs"
	refreshVault := "00000000-0000-0000-0000-000000000002"
	expiry := time.Now().Add(10 * time.Minute)
	testhelper.InsertOAuthConnectionFull(t, tx, connID, uid, "google", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		RefreshTokenVaultID: &refreshVault,
		TokenExpiry:         &expiry,
		Scopes:              []string{"openid"},
	})

	before, err := db.GetOAuthConnectionByID(context.Background(), tx, connID)
	if err != nil || before == nil {
		t.Fatalf("get connection: %v", before)
	}

	testhelper.MustExec(t, tx,
		`UPDATE oauth_connections SET status = $1, updated_at = now() WHERE id = $2`,
		db.OAuthStatusNeedsReauth, connID,
	)

	fresh, skip, err := db.ReloadOAuthConnectionIfConcurrentRefreshSucceeded(context.Background(), tx, connID, uid, before.AccessTokenVaultID, before.TokenExpiry)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if skip {
		t.Fatalf("expected skip=false when concurrent path set needs_reauth, got skip=true fresh=%+v", fresh)
	}
	if fresh == nil || fresh.Status != db.OAuthStatusNeedsReauth {
		t.Fatalf("expected fresh needs_reauth row, got %+v", fresh)
	}
}
