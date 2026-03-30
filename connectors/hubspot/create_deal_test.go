package hubspot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
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

	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()

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
	mu.Lock()
	callCount := len(calls)
	callsCopy := append([]string{}, calls...)
	mu.Unlock()
	if callCount != 3 {
		t.Errorf("expected 3 API calls, got %d: %v", callCount, callsCopy)
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

func TestCreateDeal_InvalidContactID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDealAction{conn: conn}

	params, _ := json.Marshal(createDealParams{
		DealName:           "Big Deal",
		Pipeline:           "default",
		DealStage:          "appointmentscheduled",
		AssociatedContacts: []string{"501", "../../../admin"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_deal",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric contact ID")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateDeal_TooManyAssociations(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDealAction{conn: conn}

	contacts := make([]string, 51)
	for i := range contacts {
		contacts[i] = fmt.Sprintf("%d", 100+i)
	}

	params, _ := json.Marshal(createDealParams{
		DealName:           "Big Deal",
		Pipeline:           "default",
		DealStage:          "appointmentscheduled",
		AssociatedContacts: contacts,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_deal",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for too many associations")
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
