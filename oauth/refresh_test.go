package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRefreshTokens_Success(t *testing.T) {
	t.Parallel()

	// Set up a mock OAuth token endpoint.
	expiry := time.Now().Add(1 * time.Hour).Truncate(time.Second)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	provider := Provider{
		ID:           "test-provider",
		TokenURL:     srv.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	result, err := RefreshTokens(context.Background(), provider, "old-refresh-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.AccessToken != "new-access-token" {
		t.Errorf("expected access_token 'new-access-token', got %q", result.AccessToken)
	}
	if result.RefreshToken != "new-refresh-token" {
		t.Errorf("expected refresh_token 'new-refresh-token', got %q", result.RefreshToken)
	}
	if result.Expiry.Before(expiry.Add(-5 * time.Second)) {
		t.Errorf("expected expiry around %v, got %v", expiry, result.Expiry)
	}
}

func TestRefreshTokens_KeepsOriginalRefreshToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Provider does NOT return a new refresh token.
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	provider := Provider{
		ID:           "test-provider",
		TokenURL:     srv.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	result, err := RefreshTokens(context.Background(), provider, "original-refresh-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RefreshToken != "original-refresh-token" {
		t.Errorf("expected original refresh token to be preserved, got %q", result.RefreshToken)
	}
}

func TestRefreshTokens_NoClientCredentials(t *testing.T) {
	t.Parallel()

	provider := Provider{
		ID:       "no-creds",
		TokenURL: "https://example.com/token",
	}

	_, err := RefreshTokens(context.Background(), provider, "some-token")
	if err == nil {
		t.Fatal("expected error when provider has no client credentials")
	}
	if !strings.Contains(err.Error(), "no client credentials") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRefreshTokens_EmptyRefreshToken(t *testing.T) {
	t.Parallel()

	provider := Provider{
		ID:           "test",
		TokenURL:     "https://example.com/token",
		ClientID:     "id",
		ClientSecret: "secret",
	}

	_, err := RefreshTokens(context.Background(), provider, "")
	if err == nil {
		t.Fatal("expected error when refresh token is empty")
	}
	if !strings.Contains(err.Error(), "refresh token is empty") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRefreshTokens_EmptyAccessTokenResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	provider := Provider{
		ID:           "test",
		TokenURL:     srv.URL,
		ClientID:     "id",
		ClientSecret: "secret",
	}

	_, err := RefreshTokens(context.Background(), provider, "some-token")
	if err == nil {
		t.Fatal("expected error when provider returns empty access token")
	}
	// The oauth2 library or our validation should catch this — either message is acceptable.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "empty access token") && !strings.Contains(errMsg, "missing access_token") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRefreshTokens_TokenEndpointError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "Token has been revoked",
		})
	}))
	defer srv.Close()

	provider := Provider{
		ID:           "test",
		TokenURL:     srv.URL,
		ClientID:     "id",
		ClientSecret: "secret",
	}

	_, err := RefreshTokens(context.Background(), provider, "revoked-token")
	if err == nil {
		t.Fatal("expected error when token endpoint returns error")
	}
	if !strings.Contains(err.Error(), "refresh token exchange") {
		t.Errorf("unexpected error message: %v", err)
	}
}
