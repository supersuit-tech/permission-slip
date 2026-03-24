package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetAnalytics_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/analytics/v2/reports/deals/monthly" {
			t.Errorf("expected path /analytics/v2/reports/deals/monthly, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("start") != "2026-01-01" {
			t.Errorf("expected start query param 2026-01-01, got %q", r.URL.Query().Get("start"))
		}
		if r.URL.Query().Get("end") != "2026-03-01" {
			t.Errorf("expected end query param 2026-03-01, got %q", r.URL.Query().Get("end"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"breakdowns": []map[string]any{
				{"month": "2026-01", "count": 42},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getAnalyticsAction{conn: conn}

	params, _ := json.Marshal(getAnalyticsParams{
		ObjectType: "deals",
		TimePeriod: "monthly",
		Start:      "2026-01-01T00:00:00Z",
		End:        "2026-03-01T23:59:59Z",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_analytics",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data analyticsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Breakdowns) != 1 {
		t.Errorf("expected 1 breakdown, got %d", len(data.Breakdowns))
	}
}

func TestGetAnalytics_NoDateRange(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %q", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"breakdowns": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getAnalyticsAction{conn: conn}

	params, _ := json.Marshal(getAnalyticsParams{
		ObjectType: "contacts",
		TimePeriod: "total",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_analytics",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAnalytics_MissingObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getAnalyticsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"time_period": "daily"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_analytics",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing object_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetAnalytics_InvalidObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getAnalyticsAction{conn: conn}

	params, _ := json.Marshal(getAnalyticsParams{
		ObjectType: "widgets",
		TimePeriod: "daily",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_analytics",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid object_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetAnalytics_MissingTimePeriod(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getAnalyticsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"object_type": "deals"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_analytics",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing time_period")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetAnalytics_InvalidTimePeriod(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getAnalyticsAction{conn: conn}

	params, _ := json.Marshal(getAnalyticsParams{
		ObjectType: "deals",
		TimePeriod: "hourly",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_analytics",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid time_period")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
