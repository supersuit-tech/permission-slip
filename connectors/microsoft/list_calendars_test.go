package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGraphNextRelativePath(t *testing.T) {
	t.Parallel()
	base := "https://graph.microsoft.com/v1.0"
	next := "https://graph.microsoft.com/v1.0/me/calendars?$skiptoken=abc"
	got := graphNextRelativePath(base, next)
	if got != "/me/calendars?$skiptoken=abc" {
		t.Errorf("got %q", got)
	}
}

func TestListCalendars_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/me/calendars" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{"id": "cal-1", "name": "Calendar", "isDefaultCalendar": true},
			},
		})
	}))
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	items, err := c.ListCalendars(t.Context(), validCreds())
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "cal-1" || items[0].Name != "Calendar" || !items[0].IsDefaultCalendar {
		t.Errorf("unexpected item: %+v", items[0])
	}
}
