package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAppReviewLogin_DisabledWhenNotConfigured(t *testing.T) {
	// Clear env vars to ensure the feature is disabled.
	t.Setenv("APP_REVIEW_EMAIL", "")
	t.Setenv("APP_REVIEW_OTP", "")

	deps := &Deps{}
	handler := handleAppReviewLogin(deps)

	body, _ := json.Marshal(map[string]string{
		"email": "test@example.com",
		"otp":   "123456",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/app-review-login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when not configured, got %d", rr.Code)
	}
}

func TestHandleAppReviewLogin_RejectsWrongCredentials(t *testing.T) {
	t.Setenv("APP_REVIEW_EMAIL", "review@example.com")
	t.Setenv("APP_REVIEW_OTP", "999999")

	deps := &Deps{}
	handler := handleAppReviewLogin(deps)

	tests := []struct {
		name  string
		email string
		otp   string
	}{
		{"wrong email", "wrong@example.com", "999999"},
		{"wrong otp", "review@example.com", "000000"},
		{"both wrong", "wrong@example.com", "000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"email": tt.email,
				"otp":   tt.otp,
			})
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/app-review-login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for %s, got %d", tt.name, rr.Code)
			}
		})
	}
}

func TestHandleAppReviewLogin_RejectsInvalidBody(t *testing.T) {
	t.Setenv("APP_REVIEW_EMAIL", "review@example.com")
	t.Setenv("APP_REVIEW_OTP", "999999")

	deps := &Deps{}
	handler := handleAppReviewLogin(deps)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/app-review-login", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid body, got %d", rr.Code)
	}
}

func TestHandleAppReviewLogin_EmailCaseInsensitive(t *testing.T) {
	t.Setenv("APP_REVIEW_EMAIL", "Review@Example.com")
	t.Setenv("APP_REVIEW_OTP", "999999")

	deps := &Deps{}
	handler := handleAppReviewLogin(deps)

	// Email matches case-insensitively but Supabase isn't configured,
	// so it should get past credential validation and fail at the
	// Supabase step (500) rather than the credential step (401).
	body, _ := json.Marshal(map[string]string{
		"email": "review@example.com",
		"otp":   "999999",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/app-review-login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should fail at the Supabase step (no URL configured), not at credential validation.
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 (Supabase not configured), got %d", rr.Code)
	}
}
