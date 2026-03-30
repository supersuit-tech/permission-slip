package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetInstagramInsights_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/ig_123/insights" {
			t.Errorf("expected path /ig_123/insights, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("metric") != "reach" {
			t.Errorf("expected metric=reach, got %q", r.URL.Query().Get("metric"))
		}
		if r.URL.Query().Get("period") != "week" {
			t.Errorf("expected period=week, got %q", r.URL.Query().Get("period"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(insightsResponse{
			Data: []insightData{
				{
					Name:   "reach",
					Period: "week",
					Title:  "Reach",
					Values: []insightValue{
						{Value: 1500, EndTime: "2026-03-07T08:00:00+0000"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getInstagramInsightsAction{conn: conn}

	params, _ := json.Marshal(getInstagramInsightsParams{
		InstagramAccountID: "ig_123",
		Metric:             "reach",
		Period:             "week",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_instagram_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp insightsResponse
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 insight, got %d", len(resp.Data))
	}
	if resp.Data[0].Name != "reach" {
		t.Errorf("expected metric name 'reach', got %q", resp.Data[0].Name)
	}
}

func TestGetInstagramInsights_DefaultMetricAndPeriod(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("metric") != "impressions" {
			t.Errorf("expected default metric=impressions, got %q", r.URL.Query().Get("metric"))
		}
		if r.URL.Query().Get("period") != "day" {
			t.Errorf("expected default period=day, got %q", r.URL.Query().Get("period"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(insightsResponse{Data: []insightData{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getInstagramInsightsAction{conn: conn}

	params, _ := json.Marshal(getInstagramInsightsParams{
		InstagramAccountID: "ig_123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_instagram_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetInstagramInsights_MissingAccountID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getInstagramInsightsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_instagram_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing instagram_account_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetInstagramInsights_InvalidMetric(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getInstagramInsightsAction{conn: conn}

	params, _ := json.Marshal(getInstagramInsightsParams{
		InstagramAccountID: "ig_123",
		Metric:             "invalid_metric",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_instagram_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid metric")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetInstagramInsights_InvalidPeriod(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getInstagramInsightsAction{conn: conn}

	params, _ := json.Marshal(getInstagramInsightsParams{
		InstagramAccountID: "ig_123",
		Period:             "month",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.get_instagram_insights",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
