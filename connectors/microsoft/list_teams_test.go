package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListTeams_Success(t *testing.T) {
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
					"id":          "team-1",
					"displayName": "Engineering",
					"description": "Engineering team",
					"visibility":  "private",
				},
				{
					"id":          "team-2",
					"displayName": "Marketing",
					"description": "Marketing team",
					"visibility":  "public",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listTeamsAction{conn: conn}

	params, _ := json.Marshal(listTeamsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_teams",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []teamSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(summaries))
	}
	if summaries[0].ID != "team-1" {
		t.Errorf("expected id 'team-1', got %q", summaries[0].ID)
	}
	if summaries[0].Name != "Engineering" {
		t.Errorf("expected name 'Engineering', got %q", summaries[0].Name)
	}
	if summaries[0].Visibility != "private" {
		t.Errorf("expected visibility 'private', got %q", summaries[0].Visibility)
	}
}

func TestListTeams_DefaultParams(t *testing.T) {
	t.Parallel()

	var params listTeamsParams
	params.defaults()
	if params.Top != 20 {
		t.Errorf("expected default top 20, got %d", params.Top)
	}
}

func TestListTeams_TopClamped(t *testing.T) {
	t.Parallel()

	params := listTeamsParams{Top: 100}
	params.defaults()
	if params.Top != 50 {
		t.Errorf("expected top clamped to 50, got %d", params.Top)
	}
}

func TestListTeams_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listTeamsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_teams",
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
