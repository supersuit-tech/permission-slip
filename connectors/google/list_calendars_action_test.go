package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListCalendarsAction_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/me/calendarList" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":          "primary",
					"summary":     "Personal",
					"primary":     true,
					"accessRole":  "owner",
					"timeZone":    "America/New_York",
					"description": "My personal calendar",
				},
				{
					"id":         "work@group.calendar.google.com",
					"summary":    "Work",
					"accessRole": "writer",
					"timeZone":   "America/Chicago",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &listCalendarsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendars",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validGoogleCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Calendars []calendarEntry `json:"calendars"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Calendars) != 2 {
		t.Fatalf("expected 2 calendars, got %d", len(data.Calendars))
	}

	c0 := data.Calendars[0]
	if c0.ID != "primary" || c0.Summary != "Personal" || !c0.Primary {
		t.Errorf("calendar 0: %+v", c0)
	}
	if c0.AccessRole != "owner" || c0.TimeZone != "America/New_York" || c0.Description != "My personal calendar" {
		t.Errorf("calendar 0 extra fields: %+v", c0)
	}

	c1 := data.Calendars[1]
	if c1.ID != "work@group.calendar.google.com" || c1.Summary != "Work" || c1.Primary {
		t.Errorf("calendar 1: %+v", c1)
	}
}

func TestListCalendarsAction_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []any{}})
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &listCalendarsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendars",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validGoogleCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Calendars []calendarEntry `json:"calendars"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Calendars) != 0 {
		t.Errorf("expected 0 calendars, got %d", len(data.Calendars))
	}
}

func TestListCalendarsAction_Pagination(t *testing.T) {
	t.Parallel()

	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page++
		if page == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items":         []map[string]any{{"id": "cal-1", "summary": "Calendar 1"}},
				"nextPageToken": "token-abc",
			})
		} else {
			if r.URL.Query().Get("pageToken") != "token-abc" {
				t.Errorf("expected pageToken token-abc, got %q", r.URL.Query().Get("pageToken"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{"id": "cal-2", "summary": "Calendar 2"}},
			})
		}
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &listCalendarsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendars",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validGoogleCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Calendars []calendarEntry `json:"calendars"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Calendars) != 2 {
		t.Fatalf("expected 2 calendars across pages, got %d", len(data.Calendars))
	}
}

func TestListCalendarsAction_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &listCalendarsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendars",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validGoogleCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestListCalendarsAction_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listCalendarsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendars",
		Parameters:  []byte(`{invalid`),
		Credentials: validGoogleCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON parameters")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListCalendarsAction_SummaryOverrideFallback(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"id": "cal-id", "summaryOverride": "Custom Name"},
			},
		})
	}))
	defer srv.Close()

	conn := newCalendarForTest(srv.Client(), srv.URL)
	action := &listCalendarsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_calendars",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validGoogleCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Calendars []calendarEntry `json:"calendars"`
	}
	_ = json.Unmarshal(result.Data, &data)
	if data.Calendars[0].Summary != "Custom Name" {
		t.Errorf("expected summaryOverride fallback, got %q", data.Calendars[0].Summary)
	}
}
