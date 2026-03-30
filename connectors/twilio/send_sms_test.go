package twilio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendSMS_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/Accounts/"+testAccountSID+"/Messages.json" {
			t.Errorf("path = %s, want /Accounts/%s/Messages.json", got, testAccountSID)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != testAccountSID || pass != testAuthToken {
			t.Errorf("BasicAuth = (%q, %q, %v), want (%q, %q, true)", user, pass, ok, testAccountSID, testAuthToken)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.PostFormValue("To"); got != "+15551234567" {
			t.Errorf("To = %q, want %q", got, "+15551234567")
		}
		if got := r.PostFormValue("From"); got != "+15559876543" {
			t.Errorf("From = %q, want %q", got, "+15559876543")
		}
		if got := r.PostFormValue("Body"); got != "Hello from test" {
			t.Errorf("Body = %q, want %q", got, "Hello from test")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"sid":    "SM1234567890abcdef1234567890abcdef",
			"status": "queued",
			"to":     "+15551234567",
			"from":   "+15559876543",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.send_sms"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.send_sms",
		Parameters:  json.RawMessage(`{"to":"+15551234567","from":"+15559876543","body":"Hello from test"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["sid"] != "SM1234567890abcdef1234567890abcdef" {
		t.Errorf("sid = %v, want SM1234567890abcdef1234567890abcdef", data["sid"])
	}
	if data["status"] != "queued" {
		t.Errorf("status = %v, want queued", data["status"])
	}
}

func TestSendSMS_WithMediaURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.PostFormValue("MediaUrl"); got != "https://example.com/image.jpg" {
			t.Errorf("MediaUrl = %q, want %q", got, "https://example.com/image.jpg")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"sid":    "MM1234567890abcdef1234567890abcdef",
			"status": "queued",
			"to":     "+15551234567",
			"from":   "+15559876543",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.send_sms"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.send_sms",
		Parameters:  json.RawMessage(`{"to":"+15551234567","from":"+15559876543","body":"Check this out","media_url":"https://example.com/image.jpg"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSendSMS_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["twilio.send_sms"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing to", params: `{"from":"+15559876543","body":"Hello"}`},
		{name: "missing from", params: `{"to":"+15551234567","body":"Hello"}`},
		{name: "missing body", params: `{"to":"+15551234567","from":"+15559876543"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "twilio.send_sms",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestSendSMS_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    20500,
			"message": "Internal Server Error",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.send_sms"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.send_sms",
		Parameters:  json.RawMessage(`{"to":"+15551234567","from":"+15559876543","body":"Hello"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestSendSMS_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    20003,
			"message": "Authentication Error",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.send_sms"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.send_sms",
		Parameters:  json.RawMessage(`{"to":"+15551234567","from":"+15559876543","body":"Hello"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestSendSMS_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    20429,
			"message": "Too Many Requests",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.send_sms"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.send_sms",
		Parameters:  json.RawMessage(`{"to":"+15551234567","from":"+15559876543","body":"Hello"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 60*time.Second {
			t.Errorf("RetryAfter = %v, want 60s", rle.RetryAfter)
		}
	}
}

func TestSendSMS_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.send_sms"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "twilio.send_sms",
		Parameters:  json.RawMessage(`{"to":"+15551234567","from":"+15559876543","body":"Hello"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestSendSMS_InvalidPhoneFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["twilio.send_sms"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "to missing plus", params: `{"to":"15551234567","from":"+15559876543","body":"Hello"}`},
		{name: "to has letters", params: `{"to":"+1555ABC4567","from":"+15559876543","body":"Hello"}`},
		{name: "from invalid", params: `{"to":"+15551234567","from":"not-a-number","body":"Hello"}`},
		{name: "to starts with +0", params: `{"to":"+05551234567","from":"+15559876543","body":"Hello"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "twilio.send_sms",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestSendSMS_BodyTooLong(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["twilio.send_sms"]

	params := `{"to":"+15551234567","from":"+15559876543","body":"` + strings.Repeat("a", 1601) + `"}`

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.send_sms",
		Parameters:  json.RawMessage(params),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
