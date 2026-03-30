package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRunReport_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/analytics/reports/00Oxx0000001abc" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query params, got %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"reportMetadata": map[string]any{"id": "00Oxx0000001abc", "name": "Q1 Pipeline"},
			"factMap":        map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &runReportAction{conn: conn}

	params, _ := json.Marshal(runReportParams{ReportID: "00Oxx0000001abc"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.run_report",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if _, ok := data["reportMetadata"]; !ok {
		t.Error("expected reportMetadata in result")
	}
}

func TestRunReport_WithDetails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "includeDetails=true" {
			t.Errorf("expected includeDetails=true query param, got %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"reportMetadata": map[string]any{}, "factMap": map[string]any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &runReportAction{conn: conn}

	params, _ := json.Marshal(runReportParams{
		ReportID:       "00Oxx0000001abc",
		IncludeDetails: true,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.run_report",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunReport_MissingReportID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &runReportAction{conn: conn}

	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.run_report",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing report_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestRunReport_InvalidReportID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &runReportAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"report_id": "not-valid"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.run_report",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid report_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
