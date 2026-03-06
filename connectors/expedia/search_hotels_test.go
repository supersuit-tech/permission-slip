package expedia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchHotels_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/v3/properties/availability" {
			t.Errorf("path = %s, want /v3/properties/availability", got)
		}
		q := r.URL.Query()
		if got := q.Get("checkin"); got != "2024-06-15" {
			t.Errorf("checkin = %q, want %q", got, "2024-06-15")
		}
		if got := q.Get("checkout"); got != "2024-06-17" {
			t.Errorf("checkout = %q, want %q", got, "2024-06-17")
		}
		if got := q.Get("occupancy"); got != "2" {
			t.Errorf("occupancy = %q, want %q", got, "2")
		}
		if got := q.Get("region_id"); got != "178248" {
			t.Errorf("region_id = %q, want %q", got, "178248")
		}
		if got := q.Get("currency"); got != "USD" {
			t.Errorf("currency = %q, want %q", got, "USD")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{
			{"property_id": "12345", "name": "Test Hotel", "price": 199.99},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.search_hotels"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.search_hotels",
		Parameters:  json.RawMessage(`{"checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"2","region_id":"178248","currency":"USD"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("got %d results, want 1", len(data))
	}
	if data[0]["property_id"] != "12345" {
		t.Errorf("property_id = %v, want 12345", data[0]["property_id"])
	}
}

func TestSearchHotels_LatLong(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("latitude"); got != "40.7128" {
			t.Errorf("latitude = %q, want %q", got, "40.7128")
		}
		if got := q.Get("longitude"); got != "-74.006" {
			t.Errorf("longitude = %q, want %q", got, "-74.006")
		}
		if q.Get("region_id") != "" {
			t.Error("region_id should not be set when using lat/long")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.search_hotels"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.search_hotels",
		Parameters:  json.RawMessage(`{"checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"2","latitude":40.7128,"longitude":-74.006}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchHotels_OptionalParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("sort_by"); got != "price" {
			t.Errorf("sort_by = %q, want %q", got, "price")
		}
		if got := q.Get("star_rating"); got != "4,5" {
			t.Errorf("star_rating = %q, want %q", got, "4,5")
		}
		if got := q.Get("limit"); got != "10" {
			t.Errorf("limit = %q, want %q", got, "10")
		}
		if got := q.Get("language"); got != "en-US" {
			t.Errorf("language = %q, want %q", got, "en-US")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.search_hotels"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.search_hotels",
		Parameters:  json.RawMessage(`{"checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"2","region_id":"178248","sort_by":"price","star_rating":[4,5],"limit":10,"language":"en-US"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchHotels_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["expedia.search_hotels"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing checkin",
			params: `{"checkout":"2024-06-17","occupancy":"2","region_id":"178248"}`,
		},
		{
			name:   "missing checkout",
			params: `{"checkin":"2024-06-15","occupancy":"2","region_id":"178248"}`,
		},
		{
			name:   "missing occupancy",
			params: `{"checkin":"2024-06-15","checkout":"2024-06-17","region_id":"178248"}`,
		},
		{
			name:   "missing location",
			params: `{"checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"2"}`,
		},
		{
			name:   "latitude without longitude",
			params: `{"checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"2","latitude":40.7}`,
		},
		{
			name:   "invalid checkin date",
			params: `{"checkin":"06/15/2024","checkout":"2024-06-17","occupancy":"2","region_id":"178248"}`,
		},
		{
			name:   "invalid checkout date",
			params: `{"checkin":"2024-06-15","checkout":"not-a-date","occupancy":"2","region_id":"178248"}`,
		},
		{
			name:   "checkout before checkin",
			params: `{"checkin":"2024-06-17","checkout":"2024-06-15","occupancy":"2","region_id":"178248"}`,
		},
		{
			name:   "invalid occupancy format",
			params: `{"checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"two adults","region_id":"178248"}`,
		},
		{
			name:   "invalid sort_by",
			params: `{"checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"2","region_id":"178248","sort_by":"invalid"}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "expedia.search_hotels",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestSearchHotels_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "unknown_internal_error",
			"message": "Internal server error",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["expedia.search_hotels"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "expedia.search_hotels",
		Parameters:  json.RawMessage(`{"checkin":"2024-06-15","checkout":"2024-06-17","occupancy":"2","region_id":"178248"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
