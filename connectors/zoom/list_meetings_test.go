package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListMeetings_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/me/meetings" {
			t.Errorf("expected path /users/me/meetings, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "upcoming" {
			t.Errorf("expected type=upcoming, got %s", r.URL.Query().Get("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zoomMeetingsResponse{
			Meetings: []zoomMeetingSummary{
				{
					ID:        123456,
					UUID:      "uuid-1",
					Topic:     "Team Standup",
					Type:      2,
					StartTime: "2024-01-15T09:00:00Z",
					Duration:  30,
					Timezone:  "America/New_York",
					JoinURL:   "https://zoom.us/j/123456",
					Status:    "waiting",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listMeetingsAction{conn: conn}

	params, _ := json.Marshal(listMeetingsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_meetings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		TotalMeetings int               `json:"total_meetings"`
		Meetings      []meetingListItem `json:"meetings"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Meetings) != 1 {
		t.Fatalf("expected 1 meeting, got %d", len(data.Meetings))
	}
	if data.Meetings[0].Topic != "Team Standup" {
		t.Errorf("expected topic 'Team Standup', got %q", data.Meetings[0].Topic)
	}
	if data.Meetings[0].JoinURL != "https://zoom.us/j/123456" {
		t.Errorf("expected join_url 'https://zoom.us/j/123456', got %q", data.Meetings[0].JoinURL)
	}
}

func TestListMeetings_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zoomMeetingsResponse{Meetings: nil})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listMeetingsAction{conn: conn}

	params, _ := json.Marshal(listMeetingsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_meetings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		TotalMeetings int               `json:"total_meetings"`
		Meetings      []meetingListItem `json:"meetings"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Meetings) != 0 {
		t.Errorf("expected 0 meetings, got %d", len(data.Meetings))
	}
}

func TestListMeetings_InvalidType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listMeetingsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"type": "invalid"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_meetings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListMeetings_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(zoomAPIError{Code: 124, Message: "Invalid access token."})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listMeetingsAction{conn: conn}

	params, _ := json.Marshal(listMeetingsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_meetings",
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

func TestListMeetings_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listMeetingsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_meetings",
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
