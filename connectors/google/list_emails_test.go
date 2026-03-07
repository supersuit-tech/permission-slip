package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListEmails_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")

		// Route on the URL path: /messages is the list call,
		// /messages/{id} is the get-metadata call.
		switch r.URL.Path {
		case "/gmail/v1/users/me/messages":
			json.NewEncoder(w).Encode(gmailListResponse{
				Messages: []struct {
					ID       string `json:"id"`
					ThreadID string `json:"threadId"`
				}{
					{ID: "msg-1", ThreadID: "thread-1"},
				},
				ResultSizeEstimate: 1,
			})
		default:
			// messages.get call for individual message
			json.NewEncoder(w).Encode(gmailMessageResponse{
				ID:       "msg-1",
				ThreadID: "thread-1",
				Snippet:  "Hello there",
				Payload: struct {
					Headers []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"headers"`
				}{
					Headers: []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					}{
						{Name: "From", Value: "sender@example.com"},
						{Name: "Subject", Value: "Test Email"},
					},
				},
			})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, "", "")
	action := &listEmailsAction{conn: conn}

	params, _ := json.Marshal(listEmailsParams{MaxResults: 5})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_emails",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if _, ok := data["emails"]; !ok {
		t.Error("result missing 'emails' key")
	}
}

func TestListEmails_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailListResponse{
			Messages:           nil,
			ResultSizeEstimate: 0,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, "", "")
	action := &listEmailsAction{conn: conn}

	params, _ := json.Marshal(listEmailsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_emails",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Emails []emailSummary `json:"emails"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Emails) != 0 {
		t.Errorf("expected 0 emails, got %d", len(data.Emails))
	}
}

func TestListEmails_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, "", "")
	action := &listEmailsAction{conn: conn}

	params, _ := json.Marshal(listEmailsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_emails",
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

func TestListEmails_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listEmailsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_emails",
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
