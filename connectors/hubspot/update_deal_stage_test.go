package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateDealStage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/deals/201" {
			t.Errorf("expected path /crm/v3/objects/deals/201, got %s", r.URL.Path)
		}

		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Properties["dealstage"] != "closedwon" {
			t.Errorf("expected dealstage closedwon, got %q", body.Properties["dealstage"])
		}
		if body.Properties["closedate"] != "2026-03-15" {
			t.Errorf("expected closedate 2026-03-15, got %q", body.Properties["closedate"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "201",
			"properties": map[string]string{
				"dealstage": "closedwon",
				"closedate": "2026-03-15",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateDealStageAction{conn: conn}

	params, _ := json.Marshal(updateDealStageParams{
		DealID:        "201",
		PipelineStage: "closedwon",
		CloseDate:     "2026-03-15",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_deal_stage",
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
	if data.ID != "201" {
		t.Errorf("expected id 201, got %q", data.ID)
	}
	if data.Properties["dealstage"] != "closedwon" {
		t.Errorf("expected dealstage closedwon, got %q", data.Properties["dealstage"])
	}
}

func TestUpdateDealStage_WithoutCloseDate(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if _, ok := body.Properties["closedate"]; ok {
			t.Error("expected closedate to be absent when not provided")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":         "201",
			"properties": map[string]string{"dealstage": "qualifiedtobuy"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateDealStageAction{conn: conn}

	params, _ := json.Marshal(updateDealStageParams{
		DealID:        "201",
		PipelineStage: "qualifiedtobuy",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_deal_stage",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDealStage_MissingDealID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDealStageAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"pipeline_stage": "closedwon"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_deal_stage",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing deal_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateDealStage_MissingPipelineStage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDealStageAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"deal_id": "201"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_deal_stage",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing pipeline_stage")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateDealStage_PathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDealStageAction{conn: conn}

	params, _ := json.Marshal(updateDealStageParams{
		DealID:        "../admin",
		PipelineStage: "closedwon",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_deal_stage",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric deal_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateDealStage_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "error",
			"category": "OBJECT_NOT_FOUND",
			"message":  "Deal not found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateDealStageAction{conn: conn}

	params, _ := json.Marshal(updateDealStageParams{
		DealID:        "999",
		PipelineStage: "closedwon",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_deal_stage",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found deal")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for OBJECT_NOT_FOUND, got: %T", err)
	}
}
