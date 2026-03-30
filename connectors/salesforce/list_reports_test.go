package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListReports_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/analytics/reports" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]sfReportListItem{
			{ID: "00Oxx0000001abc", Name: "Q1 Pipeline", FolderName: "Private Reports"},
			{ID: "00Oxx0000002def", Name: "Won Deals", FolderName: "Sales Reports"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listReportsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.list_reports",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["count"] != float64(2) {
		t.Errorf("expected count 2, got %v", data["count"])
	}
	reports, ok := data["reports"].([]any)
	if !ok || len(reports) != 2 {
		t.Errorf("expected 2 reports, got %v", data["reports"])
	}
}

func TestListReports_EmptyList(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]sfReportListItem{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listReportsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.list_reports",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["count"] != float64(0) {
		t.Errorf("expected count 0, got %v", data["count"])
	}
}
