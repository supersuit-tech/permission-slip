package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListMembers_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/guilds/111/members" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"user":      map[string]string{"id": "100", "username": "alice"},
				"nick":      "Alice",
				"roles":     []string{"555"},
				"joined_at": "2024-01-01T00:00:00Z",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listMembersAction{conn: conn}

	params, _ := json.Marshal(listMembersParams{GuildID: "111"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.list_members",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Members []memberSummary `json:"members"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(data.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(data.Members))
	}
	if data.Members[0].Username != "alice" {
		t.Errorf("expected username 'alice', got %q", data.Members[0].Username)
	}
	if data.Members[0].Nick != "Alice" {
		t.Errorf("expected nick 'Alice', got %q", data.Members[0].Nick)
	}
}

func TestListMembers_InvalidLimit(t *testing.T) {
	t.Parallel()
	conn := New()
	action := &listMembersAction{conn: conn}

	params, _ := json.Marshal(listMembersParams{
		GuildID: "111",
		Limit:   2000,
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "discord.list_members",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid limit")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
