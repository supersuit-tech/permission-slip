package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendEmail_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/me/sendMail" {
			t.Errorf("expected path /me/sendMail, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token in Authorization header, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected JSON content type, got %q", got)
		}

		var body graphSendMailRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Message.Body.ContentType != "HTML" {
			t.Errorf("expected default HTML content type, got %q", body.Message.Body.ContentType)
		}
		if body.Message.Subject != "Test Subject" {
			t.Errorf("expected subject 'Test Subject', got %q", body.Message.Subject)
		}
		if len(body.Message.ToRecipients) != 1 {
			t.Fatalf("expected 1 recipient, got %d", len(body.Message.ToRecipients))
		}
		if body.Message.ToRecipients[0].EmailAddress.Address != "user@example.com" {
			t.Errorf("expected recipient user@example.com, got %q", body.Message.ToRecipients[0].EmailAddress.Address)
		}

		// sendMail returns 202 Accepted with no body.
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"user@example.com"},
		Subject: "Test Subject",
		Body:    "<p>Hello</p>",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "sent" {
		t.Errorf("expected status 'sent', got %q", data["status"])
	}
}

func TestSendEmail_WithCC(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphSendMailRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Message.CCRecipients) != 1 {
			t.Errorf("expected 1 CC recipient, got %d", len(body.Message.CCRecipients))
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "body",
		CC:      []string{"cc@example.com"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendEmail_MissingTo(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"subject": "Test",
		"body":    "body",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing to")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_MissingSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"to":   []string{"user@example.com"},
		"body": "body",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token is empty.",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "body",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestSendEmail_ForbiddenMapsToAuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "ErrorAccessDenied",
				"message": "Access is denied.",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "body",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for forbidden")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for 403, got: %T", err)
	}
}

func TestSendEmail_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "body",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
	var rlErr *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rlErr) {
		if rlErr.RetryAfter.Seconds() != 30 {
			t.Errorf("expected RetryAfter 30s, got %v", rlErr.RetryAfter)
		}
	}
}

func TestSendEmail_InvalidEmailAddress(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"to":      []string{"not-an-email"},
		"subject": "Test",
		"body":    "body",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_PlainTextBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphSendMailRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Message.Body.ContentType != "Text" {
			t.Errorf("expected content type 'Text' for plain text body, got %q", body.Message.Body.ContentType)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailAction{conn: conn}
	htmlFalse := false

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "Just plain text, no HTML here.",
		HTML:    &htmlFalse,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendEmail_HTMLExplicitTrue(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphSendMailRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Message.Body.ContentType != "HTML" {
			t.Errorf("expected content type 'HTML', got %q", body.Message.Body.ContentType)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailAction{conn: conn}
	htmlTrue := true

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "minimal",
		HTML:    &htmlTrue,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendEmail_HTMLBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphSendMailRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Message.Body.ContentType != "HTML" {
			t.Errorf("expected content type 'HTML' for HTML body, got %q", body.Message.Body.ContentType)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "No angle brackets — still HTML mode by default.",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendEmail_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.send_email",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
