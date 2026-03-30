package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListConversations_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/conversations") {
			t.Errorf("expected path starting with /conversations, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conversationsResponse{
			Type:       "conversation.list",
			TotalCount: 2,
			Conversations: []intercomConversation{
				{Type: "conversation", ID: "conv_1", State: "open"},
				{Type: "conversation", ID: "conv_2", State: "open"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listConversationsAction{conn: conn}

	params, _ := json.Marshal(listConversationsParams{State: "open"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.list_conversations",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data conversationsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.TotalCount != 2 {
		t.Errorf("expected total_count 2, got %d", data.TotalCount)
	}
}

func TestListConversations_InvalidState(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listConversationsAction{conn: conn}

	params, _ := json.Marshal(listConversationsParams{State: "pending"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.list_conversations",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
