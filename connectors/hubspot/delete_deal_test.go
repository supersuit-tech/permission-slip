package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteDeal_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/deals/456" {
			t.Errorf("expected path /crm/v3/objects/deals/456, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteDealAction{conn: conn}

	params, _ := json.Marshal(deleteDealParams{DealID: "456"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.delete_deal",
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
	if data["deal_id"] != "456" {
		t.Errorf("expected deal_id 456, got %v", data["deal_id"])
	}
	if data["archived"] != true {
		t.Errorf("expected archived=true, got %v", data["archived"])
	}
}

func TestDeleteDeal_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteDealAction{conn: conn}

	params, _ := json.Marshal(deleteDealParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.delete_deal",
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
