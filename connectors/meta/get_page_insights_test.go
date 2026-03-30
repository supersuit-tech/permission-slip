package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetPageInsights_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/123456/insights" {
			t.Errorf("expected path /123456/insights, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("metric"); got != "page_impressions" {
			t.Errorf("expected metric=page_impressions, got %q", got)
		}
		if got := r.URL.Query().Get("period"); got != "day" {
			t.Errorf("expected period=day, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pageInsightsResponse{
			Data: []pageInsightDataPoint{
				{
					ID:     "123456/insights/page_impressions/day",
					Name:   "page_impressions",
					Period: "day",
					Title:  "Daily Total Impressions",
					Values: []pageInsightValue{
						{Value: 1000, EndTime: "2024-01-15T08:00:00+0000"},
						{Value: 1200, EndTime: "2024-01-16T08:00:00+0000"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getPageInsightsAction{conn: conn}

	params, _ := json.Marshal(getPageInsightsParams{
		PageID: "123456",
		Metric: "page_impressions",
		Period: "day",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_page_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data pageInsightsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Data) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(data.Data))
	}
	if data.Data[0].Name != "page_impressions" {
		t.Errorf("expected name 'page_impressions', got %q", data.Data[0].Name)
	}
}

func TestGetPageInsights_DefaultsApplied(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("metric"); got != "page_impressions" {
			t.Errorf("expected default metric=page_impressions, got %q", got)
		}
		if got := r.URL.Query().Get("period"); got != "day" {
			t.Errorf("expected default period=day, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pageInsightsResponse{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getPageInsightsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"page_id": "123456"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_page_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetPageInsights_MissingPageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPageInsightsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"metric": "page_impressions"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_page_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestGetPageInsights_InvalidMetric(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPageInsightsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"page_id": "123456",
		"metric":  "not_a_real_metric",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_page_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for invalid metric, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
