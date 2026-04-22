package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/oauth"
	"github.com/supersuit-tech/permission-slip/vault"
)

func TestOAuthRefreshInterval(t *testing.T) {
	// Cannot use t.Parallel() because subtests use t.Setenv.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name     string
		envValue string
		want     time.Duration
	}{
		{name: "default when unset", envValue: "", want: 10 * time.Minute},
		{name: "valid duration 5m", envValue: "5m", want: 5 * time.Minute},
		{name: "valid duration 1m (minimum)", envValue: "1m", want: time.Minute},
		{name: "valid duration 30m", envValue: "30m", want: 30 * time.Minute},
		{name: "invalid string falls back", envValue: "notaduration", want: 10 * time.Minute},
		{name: "zero duration falls back", envValue: "0s", want: 10 * time.Minute},
		{name: "negative duration falls back", envValue: "-5m", want: 10 * time.Minute},
		{name: "below minimum falls back", envValue: "30s", want: 10 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OAUTH_REFRESH_INTERVAL", tt.envValue)
			got := oauthRefreshInterval(logger)
			if got != tt.want {
				t.Errorf("oauthRefreshInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartOAuthRefresh_StopsOnContextCancel(t *testing.T) {
	// Cannot use t.Parallel() because we set OAUTH_REFRESH_INTERVAL via Setenv.
	ctx, cancel := context.WithCancel(context.Background())
	t.Setenv("OAUTH_REFRESH_INTERVAL", "1m")

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Use a nil DB/vault/registry — runOAuthRefresh will fail to list connections
	// but the goroutine lifecycle is what we're testing.
	deps := OAuthRefreshDeps{}

	done := startOAuthRefresh(ctx, deps, logger)

	// Give the goroutine time to run the initial pass.
	time.Sleep(50 * time.Millisecond)

	cancel()
	select {
	case <-done:
		// Success — goroutine exited.
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine did not exit within 2s after context cancel")
	}

	output := buf.String()
	if !strings.Contains(output, "oauth refresh: scheduled") {
		t.Errorf("expected startup log message, got: %s", output)
	}
}

func TestRefreshSingleConnection_MultiInstanceNoCrossContamination(t *testing.T) {
	t.Parallel()

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		rt := r.FormValue("refresh_token")
		w.Header().Set("Content-Type", "application/json")
		switch rt {
		case "bg-refresh-alpha":
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "bg-access-alpha",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		case "bg-refresh-beta":
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "bg-access-beta",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"})
		}
	}))
	defer tokenSrv.Close()

	v := vault.NewMockVaultStore()
	reg := oauth.NewRegistry()
	_ = reg.Register(oauth.Provider{
		ID:           "slack",
		TokenURL:     tokenSrv.URL,
		ClientID:     "id",
		ClientSecret: "secret",
		Source:       oauth.SourceBuiltIn,
	})
	deps := OAuthRefreshDeps{DB: tx, Vault: v, Registry: reg}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	refreshA := "11111111-1111-1111-1111-111111111111"
	refreshB := "22222222-2222-2222-2222-222222222222"
	v.SeedSecretForTest(refreshA, []byte("bg-refresh-alpha"))
	v.SeedSecretForTest(refreshB, []byte("bg-refresh-beta"))

	expSoon := time.Now().Add(10 * time.Minute)
	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_bg_a", uid, "slack", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		RefreshTokenVaultID: &refreshA,
		TokenExpiry:         &expSoon,
		Scopes:              []string{},
	})
	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_bg_b", uid, "slack", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID:  "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		RefreshTokenVaultID: &refreshB,
		TokenExpiry:         &expSoon,
		Scopes:              []string{},
	})
	v.SeedSecretForTest("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", []byte("old-a"))
	v.SeedSecretForTest("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", []byte("old-b"))

	connA, err := db.GetOAuthConnectionByID(t.Context(), tx, "oconn_bg_a")
	if err != nil || connA == nil {
		t.Fatalf("conn A: %v", connA)
	}
	connB, err := db.GetOAuthConnectionByID(t.Context(), tx, "oconn_bg_b")
	if err != nil || connB == nil {
		t.Fatalf("conn B: %v", connB)
	}

	if err := refreshSingleConnection(t.Context(), deps, logger, *connA); err != nil {
		t.Fatalf("refresh A: %v", err)
	}
	if err := refreshSingleConnection(t.Context(), deps, logger, *connB); err != nil {
		t.Fatalf("refresh B: %v", err)
	}

	rowA, _ := db.GetOAuthConnectionByID(t.Context(), tx, "oconn_bg_a")
	rowB, _ := db.GetOAuthConnectionByID(t.Context(), tx, "oconn_bg_b")
	if rowA == nil || rowB == nil {
		t.Fatal("reload rows")
	}
	if rowA.AccessTokenVaultID == rowB.AccessTokenVaultID {
		t.Error("connections must not share access_token_vault_id")
	}
	if rowA.RefreshTokenVaultID != nil && rowB.RefreshTokenVaultID != nil && *rowA.RefreshTokenVaultID == *rowB.RefreshTokenVaultID {
		t.Error("connections must not share refresh_token_vault_id")
	}

	atA, err := v.ReadSecret(t.Context(), tx, rowA.AccessTokenVaultID)
	if err != nil {
		t.Fatalf("read access A: %v", err)
	}
	atB, err := v.ReadSecret(t.Context(), tx, rowB.AccessTokenVaultID)
	if err != nil {
		t.Fatalf("read access B: %v", err)
	}
	if string(atA) != "bg-access-alpha" || string(atB) != "bg-access-beta" {
		t.Errorf("unexpected vault contents: %q / %q", atA, atB)
	}
}

func TestRefreshSingleConnection_InvalidGrantAfterConcurrentRefreshStaysActive(t *testing.T) {
	t.Parallel()

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	bumpCh := make(chan struct{})
	bumpDone := make(chan struct{})
	go func() {
		defer close(bumpDone)
		<-bumpCh
		newAccessID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		testhelper.MustExec(t, tx,
			`UPDATE oauth_connections SET
				token_expiry = now() + interval '2 hours',
				updated_at = now(),
				access_token_vault_id = $1,
				status = $2
			WHERE id = $3 AND user_id = $4`,
			newAccessID, db.OAuthStatusActive, "oconn_bg_race", uid,
		)
	}()

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(bumpCh)
		<-bumpDone
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "refresh token already used",
		})
	}))
	defer tokenSrv.Close()

	v := vault.NewMockVaultStore()
	v.SeedSecretForTest("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", []byte("winner-access"))

	reg := oauth.NewRegistry()
	_ = reg.Register(oauth.Provider{
		ID:           "google",
		TokenURL:     tokenSrv.URL,
		ClientID:     "id",
		ClientSecret: "secret",
		Source:       oauth.SourceBuiltIn,
	})
	deps := OAuthRefreshDeps{DB: tx, Vault: v, Registry: reg}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	refreshVault := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	v.SeedSecretForTest(refreshVault, []byte("stale-refresh"))
	expSoon := time.Now().Add(10 * time.Minute)
	testhelper.InsertOAuthConnectionFull(t, tx, "oconn_bg_race", uid, "google", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID:  "cccccccc-cccc-cccc-cccc-cccccccccccc",
		RefreshTokenVaultID: &refreshVault,
		TokenExpiry:         &expSoon,
		Scopes:              []string{},
	})
	v.SeedSecretForTest("cccccccc-cccc-cccc-cccc-cccccccccccc", []byte("stale-access"))

	conn, err := db.GetOAuthConnectionByID(t.Context(), tx, "oconn_bg_race")
	if err != nil || conn == nil {
		t.Fatalf("get conn: %v", conn)
	}

	if err := refreshSingleConnection(t.Context(), deps, logger, *conn); err != nil {
		t.Fatalf("expected success after concurrent bump, got %v", err)
	}

	updated, err := db.GetOAuthConnectionByID(t.Context(), tx, "oconn_bg_race")
	if err != nil || updated == nil {
		t.Fatalf("reload: %v", updated)
	}
	if updated.Status != db.OAuthStatusActive {
		t.Fatalf("expected active, got %q", updated.Status)
	}
}
