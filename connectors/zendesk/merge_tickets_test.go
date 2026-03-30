package zendesk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestMergeTickets_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/tickets/100/merge.json" {
			t.Errorf("expected path /tickets/100/merge.json, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(jobStatusResponse{
			JobStatus: jobStatus{ID: "job-123", Status: "queued"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &mergeTicketsAction{conn: conn}

	params, _ := json.Marshal(mergeTicketsParams{
		TargetID:  100,
		SourceIDs: []int64{101, 102},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.merge_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data jobStatusResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.JobStatus.ID != "job-123" {
		t.Errorf("expected job id 'job-123', got %q", data.JobStatus.ID)
	}
}

func TestMergeTickets_MissingTargetID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &mergeTicketsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"source_ids": []int64{1, 2}})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.merge_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing target_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestMergeTickets_SourceSameAsTarget(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &mergeTicketsAction{conn: conn}

	params, _ := json.Marshal(mergeTicketsParams{
		TargetID:  100,
		SourceIDs: []int64{100},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.merge_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when source == target")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestMergeTickets_TooManySources(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &mergeTicketsAction{conn: conn}

	params, _ := json.Marshal(mergeTicketsParams{
		TargetID:  100,
		SourceIDs: []int64{1, 2, 3, 4, 5, 6},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zendesk.merge_tickets",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for too many source_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
