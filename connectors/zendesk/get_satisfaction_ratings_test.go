package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetSatisfactionRatings_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/satisfaction_ratings.json") {
			t.Errorf("expected path /satisfaction_ratings.json, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("score") != "good" {
			t.Errorf("expected score=good, got %q", r.URL.Query().Get("score"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(satisfactionRatingsResponse{
			Count: 1,
			SatisfactionRatings: []satisfactionRating{
				{ID: 1, Score: "good", TicketID: 100},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getSatisfactionRatingsAction{conn: conn}

	params, _ := json.Marshal(getSatisfactionRatingsParams{Score: "good"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.get_satisfaction_ratings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data satisfactionRatingsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Count != 1 {
		t.Errorf("expected count 1, got %d", data.Count)
	}
	if data.SatisfactionRatings[0].Score != "good" {
		t.Errorf("expected score good, got %q", data.SatisfactionRatings[0].Score)
	}
}

func TestGetSatisfactionRatings_InvalidScore(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getSatisfactionRatingsAction{conn: conn}

	params, _ := json.Marshal(getSatisfactionRatingsParams{Score: "excellent"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.get_satisfaction_ratings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid score")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetSatisfactionRatings_NoScore(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(satisfactionRatingsResponse{Count: 0, SatisfactionRatings: []satisfactionRating{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getSatisfactionRatingsAction{conn: conn}

	params, _ := json.Marshal(getSatisfactionRatingsParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.get_satisfaction_ratings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
