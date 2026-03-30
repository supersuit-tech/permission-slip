package mobilepush

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/notify"
)

func TestBuildMessage(t *testing.T) {
	t.Parallel()
	t.Run("basic approval", func(t *testing.T) {
		t.Parallel()
		approval := notify.Approval{
			ApprovalID:  "appr_abc123",
			AgentName:   "code-reviewer",
			ApprovalURL: "https://app.test/approve/appr_abc123",
			ExpiresAt:   time.Now().Add(time.Hour),
		}

		msg := buildMessage(approval)
		if msg.Title != "code-reviewer" {
			t.Errorf("expected title 'code-reviewer', got %q", msg.Title)
		}
		if msg.ApprovalID != "appr_abc123" {
			t.Errorf("expected approval_id 'appr_abc123', got %q", msg.ApprovalID)
		}
		if msg.URL != "https://app.test/approve/appr_abc123" {
			t.Errorf("expected URL, got %q", msg.URL)
		}
	})

	t.Run("action with summary", func(t *testing.T) {
		t.Parallel()
		action := map[string]string{"type": "email.send", "summary": "Send invoice to client"}
		actionJSON, _ := json.Marshal(action)

		approval := notify.Approval{
			AgentName: "billing-bot",
			Action:    actionJSON,
		}

		msg := buildMessage(approval)
		if msg.Body != "Send invoice to client" {
			t.Errorf("expected summary body, got %q", msg.Body)
		}
	})

	t.Run("action with only type", func(t *testing.T) {
		t.Parallel()
		action := map[string]string{"type": "github.merge_pr"}
		actionJSON, _ := json.Marshal(action)

		approval := notify.Approval{
			AgentName: "deploy-bot",
			Action:    actionJSON,
		}

		msg := buildMessage(approval)
		if msg.Body != "github.merge_pr" {
			t.Errorf("expected action type as body, got %q", msg.Body)
		}
	})

	t.Run("no agent name uses default title", func(t *testing.T) {
		t.Parallel()
		approval := notify.Approval{}

		msg := buildMessage(approval)
		if msg.Title != "Approval Request" {
			t.Errorf("expected default title, got %q", msg.Title)
		}
	})

	t.Run("long summary is truncated", func(t *testing.T) {
		t.Parallel()
		longSummary := ""
		for i := 0; i < 250; i++ {
			longSummary += "x"
		}
		action := map[string]string{"summary": longSummary}
		actionJSON, _ := json.Marshal(action)

		approval := notify.Approval{
			AgentName: "test",
			Action:    actionJSON,
		}

		msg := buildMessage(approval)
		if len(msg.Body) > 200 {
			t.Errorf("expected body to be truncated to 200 chars, got %d", len(msg.Body))
		}
		if msg.Body[len(msg.Body)-3:] != "..." {
			t.Errorf("expected truncated body to end with '...', got %q", msg.Body[len(msg.Body)-3:])
		}
	})

	t.Run("payment failed notification", func(t *testing.T) {
		t.Parallel()
		approval := notify.Approval{
			Type:        notify.NotificationTypePaymentFailed,
			ApprovalURL: "https://app.test/settings/billing",
			ApprovalID:  "appr_billing",
		}

		msg := buildMessage(approval)
		if msg.Title != "Payment Failed" {
			t.Errorf("expected 'Payment Failed', got %q", msg.Title)
		}
		if msg.URL != "https://app.test/settings/billing" {
			t.Errorf("expected billing URL, got %q", msg.URL)
		}
	})
}

func TestSenderName(t *testing.T) {
	t.Parallel()
	s := &Sender{}
	if s.Name() != "mobile-push" {
		t.Errorf("expected Name() to return 'mobile-push', got %q", s.Name())
	}
}

// --- HTTP-level sender tests using httptest.NewServer ---

// mockStore implements TokenStore for sender tests without requiring
// real pgx types. For full DB integration tests, see db/expo_push_tokens_test.go.
type mockStore struct {
	mu            sync.Mutex
	tokens        []db.ExpoPushToken
	deletedTokens []string
}

func (m *mockStore) addToken(id int64, token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens = append(m.tokens, db.ExpoPushToken{
		ID:    id,
		Token: token,
	})
}

func (m *mockStore) wasDeleted(token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.deletedTokens {
		if t == token {
			return true
		}
	}
	return false
}

func (m *mockStore) ListTokens(ctx context.Context, userID string) ([]db.ExpoPushToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]db.ExpoPushToken, len(m.tokens))
	copy(result, m.tokens)
	return result, nil
}

func (m *mockStore) DeleteToken(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deletedTokens = append(m.deletedTokens, token)
	return nil
}

func testApproval() notify.Approval {
	return notify.Approval{
		ApprovalID:  "appr_test123",
		AgentName:   "test-agent",
		ApprovalURL: "https://app.test/approve/appr_test123",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
}

// newTestSender creates a Sender that uses a mockStore and custom API URL
// (bypassing db.DBTX and the production Expo endpoint).
func newTestSender(store TokenStore, accessToken, apiURL string) *Sender {
	return &Sender{
		store:       store,
		accessToken: accessToken,
		apiURL:      apiURL,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

func TestSendBatch_Success(t *testing.T) {
	t.Parallel()
	var capturedBody []byte
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expoAPIResponse{
			Data: []expoTicketResponse{
				{Status: "ok", ID: "ticket-1"},
				{Status: "ok", ID: "ticket-2"},
			},
		})
	}))
	defer server.Close()

	store := &mockStore{}
	store.addToken(1, "ExponentPushToken[device1]")
	store.addToken(2, "ExponentPushToken[device2]")

	sender := newTestSender(store, "test-access-token", server.URL)
	err := sender.Send(context.Background(), testApproval(), notify.Recipient{
		UserID:   "user-1",
		Username: "alice",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify Authorization header
	if capturedHeaders.Get("Authorization") != "Bearer test-access-token" {
		t.Errorf("expected Bearer auth header, got %q", capturedHeaders.Get("Authorization"))
	}

	// Verify Content-Type
	if capturedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json content-type, got %q", capturedHeaders.Get("Content-Type"))
	}

	// Verify batch request body contains both tokens
	var messages []expoMessage
	if err := json.Unmarshal(capturedBody, &messages); err != nil {
		t.Fatalf("failed to unmarshal request body: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages in batch, got %d", len(messages))
	}
	if messages[0].To != "ExponentPushToken[device1]" {
		t.Errorf("expected first token, got %q", messages[0].To)
	}
	if messages[1].To != "ExponentPushToken[device2]" {
		t.Errorf("expected second token, got %q", messages[1].To)
	}
}

func TestSendBatch_NoAccessToken(t *testing.T) {
	t.Parallel()
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expoAPIResponse{
			Data: []expoTicketResponse{{Status: "ok", ID: "ticket-1"}},
		})
	}))
	defer server.Close()

	store := &mockStore{}
	store.addToken(1, "ExponentPushToken[device1]")

	sender := newTestSender(store, "", server.URL) // no access token
	err := sender.Send(context.Background(), testApproval(), notify.Recipient{
		UserID:   "user-1",
		Username: "alice",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedHeaders.Get("Authorization") != "" {
		t.Errorf("expected no Authorization header, got %q", capturedHeaders.Get("Authorization"))
	}
}

func TestSendBatch_DeviceNotRegistered_CleansUpToken(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expoAPIResponse{
			Data: []expoTicketResponse{
				{
					Status:  "error",
					Message: "The recipient device is not registered with FCM",
					Details: &struct {
						Error string `json:"error"`
					}{Error: "DeviceNotRegistered"},
				},
			},
		})
	}))
	defer server.Close()

	store := &mockStore{}
	store.addToken(1, "ExponentPushToken[invalid]")

	sender := newTestSender(store, "", server.URL)
	err := sender.Send(context.Background(), testApproval(), notify.Recipient{
		UserID:   "user-1",
		Username: "alice",
	})

	// DeviceNotRegistered should not be returned as an error
	if err != nil {
		t.Fatalf("expected nil error for DeviceNotRegistered, got: %v", err)
	}

	// Token should have been deleted
	if !store.wasDeleted("ExponentPushToken[invalid]") {
		t.Error("expected invalid token to be deleted")
	}
}

func TestSendBatch_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	store := &mockStore{}
	store.addToken(1, "ExponentPushToken[device1]")

	sender := newTestSender(store, "", server.URL)
	err := sender.Send(context.Background(), testApproval(), notify.Recipient{
		UserID:   "user-1",
		Username: "alice",
	})

	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestSendBatch_TopLevelAPIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expoAPIResponse{
			Errors: []struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				{Code: "PUSH_TOO_MANY_EXPERIENCE_IDS", Message: "too many experience IDs"},
			},
		})
	}))
	defer server.Close()

	store := &mockStore{}
	store.addToken(1, "ExponentPushToken[device1]")

	sender := newTestSender(store, "", server.URL)
	err := sender.Send(context.Background(), testApproval(), notify.Recipient{
		UserID:   "user-1",
		Username: "alice",
	})

	if err == nil {
		t.Fatal("expected error for API-level error response")
	}
}

func TestCategoryForType_StandingExecution(t *testing.T) {
	t.Parallel()
	if got := categoryForType(notify.NotificationTypeStandingExecution); got != "standing_execution" {
		t.Errorf("expected 'standing_execution', got %q", got)
	}
}

func TestCategoryForType_DefaultApproval(t *testing.T) {
	t.Parallel()
	if got := categoryForType(notify.NotificationTypeApproval); got != "approval" {
		t.Errorf("expected 'approval', got %q", got)
	}
}

func TestCategoryForType_PaymentFailed(t *testing.T) {
	t.Parallel()
	if got := categoryForType(notify.NotificationTypePaymentFailed); got != "approval" {
		t.Errorf("expected 'approval' for payment_failed, got %q", got)
	}
}

func TestSendBatch_StandingExecution_UsesCorrectCategory(t *testing.T) {
	t.Parallel()
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expoAPIResponse{
			Data: []expoTicketResponse{{Status: "ok", ID: "ticket-1"}},
		})
	}))
	defer server.Close()

	store := &mockStore{}
	store.addToken(1, "ExponentPushToken[device1]")

	sender := newTestSender(store, "", server.URL)
	approval := notify.Approval{
		ApprovalID:  "appr_standing_001",
		AgentName:   "Deploy Bot",
		Action:      json.RawMessage(`{"type":"github.issues.create"}`),
		Context:     json.RawMessage(`{"execution_count":3,"max_executions":10}`),
		ApprovalURL: "https://app.test/activity",
		Type:        notify.NotificationTypeStandingExecution,
	}
	err := sender.Send(context.Background(), approval, notify.Recipient{
		UserID:   "user-1",
		Username: "alice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var messages []expoMessage
	if err := json.Unmarshal(capturedBody, &messages); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].CategoryID != "standing_execution" {
		t.Errorf("expected categoryId 'standing_execution', got %q", messages[0].CategoryID)
	}
}

func TestSend_NoTokens(t *testing.T) {
	t.Parallel()
	requestMade := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))
	defer server.Close()

	store := &mockStore{} // no tokens

	sender := newTestSender(store, "", server.URL)
	err := sender.Send(context.Background(), testApproval(), notify.Recipient{
		UserID:   "user-1",
		Username: "alice",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestMade {
		t.Error("expected no HTTP request when there are no tokens")
	}
}
