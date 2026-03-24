package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGraphNextRelativePath(t *testing.T) {
	t.Parallel()
	base := "https://graph.microsoft.com/v1.0"

	tests := []struct {
		name string
		next string
		want string
	}{
		{"valid path", "https://graph.microsoft.com/v1.0/me/calendars?$skiptoken=abc", "/me/calendars?$skiptoken=abc"},
		{"empty next", "", ""},
		{"different host", "https://evil.com/v1.0/me/calendars", ""},
		{"prefix collision without path boundary", "https://graph.microsoft.com/v1.0.evil.com/steal", ""},
		{"query string after base", "https://graph.microsoft.com/v1.0?unexpected=true", "?unexpected=true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := graphNextRelativePath(base, tt.next)
			if got != tt.want {
				t.Errorf("graphNextRelativePath(%q, %q) = %q, want %q", base, tt.next, got, tt.want)
			}
		})
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
