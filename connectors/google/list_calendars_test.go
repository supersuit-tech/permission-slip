package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListCalendars_Success(t *testing.T) {
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
				{"id": "primary", "summary": "Personal", "primary": true},
				{"id": "work@group.calendar.google.com", "summary": "Work"},
			},
		})
	}))
	defer srv.Close()

	c := newCalendarForTest(srv.Client(), srv.URL)
	items, err := c.ListCalendars(t.Context(), validGoogleCreds())
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "primary" || items[0].Summary != "Personal" || !items[0].Primary {
		t.Errorf("item 0: %+v", items[0])
	}
	if items[1].ID != "work@group.calendar.google.com" || items[1].Summary != "Work" {
		t.Errorf("item 1: %+v", items[1])
	}
}

func validGoogleCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{"access_token": "t"})
}
