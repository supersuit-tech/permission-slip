package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateAdCampaign_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/act_123456789/campaigns" {
			t.Errorf("expected path /act_123456789/campaigns, got %s", r.URL.Path)
		}

		var body createAdCampaignRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Name != "Summer Sale 2024" {
			t.Errorf("expected name 'Summer Sale 2024', got %q", body.Name)
		}
		if body.Objective != "OUTCOME_SALES" {
			t.Errorf("expected objective 'OUTCOME_SALES', got %q", body.Objective)
		}
		if body.Status != "PAUSED" {
			t.Errorf("expected default status 'PAUSED', got %q", body.Status)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createAdCampaignResponse{ID: "camp-789"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createAdCampaignAction{conn: conn}

	params, _ := json.Marshal(createAdCampaignParams{
		AdAccountID: "123456789",
		Name:        "Summer Sale 2024",
		Objective:   "OUTCOME_SALES",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_ad_campaign",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "camp-789" {
		t.Errorf("expected id 'camp-789', got %q", data["id"])
	}
}

func TestCreateAdCampaign_InvalidObjective(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createAdCampaignAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"ad_account_id": "123456789",
		"name":          "Test",
		"objective":     "INVALID_OBJECTIVE",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_ad_campaign",
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

func TestCreateAdCampaign_MissingAdAccountID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createAdCampaignAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"name":      "Test Campaign",
		"objective": "OUTCOME_SALES",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_ad_campaign",
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
