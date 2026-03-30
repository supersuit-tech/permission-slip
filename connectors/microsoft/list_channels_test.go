package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListChannels_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token in Authorization header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{
					"id":             "channel-1",
					"displayName":    "General",
					"description":    "General discussion",
					"membershipType": "standard",
				},
				{
					"id":             "channel-2",
					"displayName":    "Random",
					"description":    "Random chat",
					"membershipType": "private",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listChannelsAction{conn: conn}

	params, _ := json.Marshal(listChannelsParams{TeamID: "team-1"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []channelSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(summaries))
	}
	if summaries[0].ID != "channel-1" {
		t.Errorf("expected id 'channel-1', got %q", summaries[0].ID)
	}
	if summaries[0].Name != "General" {
		t.Errorf("expected name 'General', got %q", summaries[0].Name)
	}
	if summaries[0].MembershipType != "standard" {
		t.Errorf("expected membership_type 'standard', got %q", summaries[0].MembershipType)
	}
}

func TestListChannels_MissingTeamID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listChannelsAction{conn: conn}

	params, _ := json.Marshal(listChannelsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_channels",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing team_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListChannels_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listChannelsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_channels",
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
