package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateDeal_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/deals" {
			t.Errorf("expected path /crm/v3/objects/deals, got %s", r.URL.Path)
		}

		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Properties["dealname"] != "Big Deal" {
			t.Errorf("expected dealname 'Big Deal', got %q", body.Properties["dealname"])
		}
		if body.Properties["pipeline"] != "default" {
			t.Errorf("expected pipeline default, got %q", body.Properties["pipeline"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "601",
			"properties": body.Properties,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createDealAction{conn: conn}

	params, _ := json.Marshal(createDealParams{
		DealName:  "Big Deal",
		Pipeline:  "default",
		DealStage: "appointmentscheduled",
		Amount:    "10000",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_deal",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data hubspotObjectResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "601" {
		t.Errorf("expected id 601, got %q", data.ID)
	}
}

func TestCreateDeal_WithAssociations(t *testing.T) {
	t.Parallel()

	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)

		if r.URL.Path == "/crm/v3/objects/deals" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"id":         "601",
				"properties": map[string]string{"dealname": "Big Deal"},
			})
			return
		}

		// Association call
		if strings.Contains(r.URL.Path, "/associations/") {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT for association, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createDealAction{conn: conn}

	params, _ := json.Marshal(createDealParams{
		DealName:           "Big Deal",
		Pipeline:           "default",
		DealStage:          "appointmentscheduled",
		AssociatedContacts: []string{"501", "502"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_deal",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 create + 2 associations = 3 calls
	if len(calls) != 3 {
		t.Errorf("expected 3 API calls, got %d: %v", len(calls), calls)
	}
}

func TestCreateDeal_MissingDealName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDealAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"pipeline":  "default",
		"dealstage": "appointmentscheduled",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_deal",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing dealname")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateDeal_MissingPipeline(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDealAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"dealname":  "Big Deal",
		"dealstage": "appointmentscheduled",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_deal",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing pipeline")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateDeal_MissingDealStage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDealAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"dealname": "Big Deal",
		"pipeline": "default",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_deal",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing dealstage")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
