package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetCampaignStats_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/marketing/singlesends/ss_123" {
			t.Errorf("path = %s, want /marketing/singlesends/ss_123", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "ss_123",
			"name":   "March Newsletter",
			"status": "triggered",
			"send_stats": map[string]any{
				"requests":      1000,
				"delivered":     980,
				"opens":         450,
				"unique_opens":  400,
				"clicks":        120,
				"unique_clicks": 100,
				"bounces":       20,
				"spam_reports":  2,
				"unsubscribes":  5,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.get_campaign_stats"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.get_campaign_stats",
		Parameters:  json.RawMessage(`{"singlesend_id":"ss_123"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["singlesend_id"] != "ss_123" {
		t.Errorf("singlesend_id = %v, want ss_123", data["singlesend_id"])
	}
	if data["status"] != "triggered" {
		t.Errorf("status = %v, want triggered", data["status"])
	}
	stats, ok := data["stats"].(map[string]any)
	if !ok {
		t.Fatal("stats not present in result")
	}
	if stats["delivered"] != float64(980) {
		t.Errorf("stats.delivered = %v, want 980", stats["delivered"])
	}
	if stats["opens"] != float64(450) {
		t.Errorf("stats.opens = %v, want 450", stats["opens"])
	}
}

func TestGetCampaignStats_NoStats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "ss_draft",
			"name":   "Draft Campaign",
			"status": "draft",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.get_campaign_stats"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.get_campaign_stats",
		Parameters:  json.RawMessage(`{"singlesend_id":"ss_draft"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if _, ok := data["stats"]; ok {
		t.Error("expected no stats for draft campaign")
	}
}

func TestGetCampaignStats_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.get_campaign_stats"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing singlesend_id", params: `{}`},
		{name: "empty singlesend_id", params: `{"singlesend_id":""}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.get_campaign_stats",
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
