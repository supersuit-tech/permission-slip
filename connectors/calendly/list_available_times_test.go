package calendly

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListAvailableTimes_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/event_type_available_times" {
			t.Errorf("expected path /event_type_available_times, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("event_type") == "" {
			t.Error("expected event_type query param")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(calendlyAvailableTimesResponse{
			Collection: []calendlyAvailableTime{
				{
					Status:            "available",
					InviteesRemaining: 1,
					StartTime:         "2024-01-15T09:00:00.000000Z",
					SchedulingURL:     "https://calendly.com/testuser/30min?month=2024-01&date=2024-01-15",
				},
				{
					Status:            "available",
					InviteesRemaining: 1,
					StartTime:         "2024-01-15T10:00:00.000000Z",
					SchedulingURL:     "https://calendly.com/testuser/30min?month=2024-01&date=2024-01-15",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listAvailableTimesAction{conn: conn}

	params, _ := json.Marshal(listAvailableTimesParams{
		EventTypeURI: "https://api.calendly.com/event_types/et1",
		StartTime:    "2024-01-15T00:00:00Z",
		EndTime:      "2024-01-16T00:00:00Z",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_available_times",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		TotalAvailableTimes int                 `json:"total_available_times"`
		AvailableTimes      []calendlyAvailableTime `json:"available_times"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.AvailableTimes) != 2 {
		t.Fatalf("expected 2 available times, got %d", len(data.AvailableTimes))
	}
	if data.AvailableTimes[0].Status != "available" {
		t.Errorf("expected status 'available', got %q", data.AvailableTimes[0].Status)
	}
}

func TestListAvailableTimes_MissingRequiredParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params listAvailableTimesParams
	}{
		{
			name:   "missing event_type_uri",
			params: listAvailableTimesParams{StartTime: "2024-01-15T00:00:00Z", EndTime: "2024-01-16T00:00:00Z"},
		},
		{
			name:   "missing start_time",
			params: listAvailableTimesParams{EventTypeURI: "https://api.calendly.com/event_types/et1", EndTime: "2024-01-16T00:00:00Z"},
		},
		{
			name:   "missing end_time",
			params: listAvailableTimesParams{EventTypeURI: "https://api.calendly.com/event_types/et1", StartTime: "2024-01-15T00:00:00Z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := New()
			action := &listAvailableTimesAction{conn: conn}

			params, _ := json.Marshal(tt.params)

			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "calendly.list_available_times",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error for missing required param")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestListAvailableTimes_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listAvailableTimesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "calendly.list_available_times",
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
