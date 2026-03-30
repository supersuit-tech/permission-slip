package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestGetAgentPaymentMethod_NoAssignment(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "apmapi_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/payment-method", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["payment_method_id"] != nil {
		t.Errorf("expected null payment_method_id, got %v", resp["payment_method_id"])
	}
}

func TestAssignAgentPaymentMethod(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "apmassign_"+uid[:8])

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_api_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "4242",
		ExpMonth:              12,
		ExpYear:               2028,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body, _ := json.Marshal(map[string]string{"payment_method_id": pm.ID})
	r := authenticatedRequestWithBody(t, http.MethodPut, fmt.Sprintf("/agents/%d/payment-method", agentID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentPaymentMethodResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.PaymentMethodID != pm.ID {
		t.Errorf("expected payment_method_id=%s, got %s", pm.ID, resp.PaymentMethodID)
	}
	if resp.AgentID != agentID {
		t.Errorf("expected agent_id=%d, got %d", agentID, resp.AgentID)
	}
}

func TestAssignAgentPaymentMethod_OtherUserPM(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	otherUID := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "apmown_"+uid[:8])
	testhelper.InsertUser(t, tx, otherUID, "apmoth_"+otherUID[:8])

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                otherUID,
		StripePaymentMethodID: "pm_other_" + otherUID[:8],
		Brand:                 "visa",
		Last4:                 "9999",
		ExpMonth:              12,
		ExpYear:               2028,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	body, _ := json.Marshal(map[string]string{"payment_method_id": pm.ID})
	r := authenticatedRequestWithBody(t, http.MethodPut, fmt.Sprintf("/agents/%d/payment-method", agentID), uid, body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for other user's PM, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveAgentPaymentMethod(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "apmrm_"+uid[:8])

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_rm_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "4242",
		ExpMonth:              12,
		ExpYear:               2028,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	_, err = db.AssignAgentPaymentMethod(ctx, tx, agentID, pm.ID)
	if err != nil {
		t.Fatalf("AssignAgentPaymentMethod: %v", err)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/agents/%d/payment-method", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone.
	binding, err := db.GetAgentPaymentMethod(ctx, tx, agentID)
	if err != nil {
		t.Fatalf("GetAgentPaymentMethod: %v", err)
	}
	if binding != nil {
		t.Fatal("expected nil after removal")
	}
}

func TestRemoveAgentPaymentMethod_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "apmnf_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/agents/%d/payment-method", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeletePaymentMethod_ReturnsAffectedAgents(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmaffect_"+uid[:8])
	agentID1 := testhelper.InsertAgent(t, tx, uid)
	agentID2 := testhelper.InsertAgent(t, tx, uid)

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_affect_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "4242",
		ExpMonth:              12,
		ExpYear:               2028,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	_, _ = db.AssignAgentPaymentMethod(ctx, tx, agentID1, pm.ID)
	_, _ = db.AssignAgentPaymentMethod(ctx, tx, agentID2, pm.ID)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, fmt.Sprintf("/payment-methods/%s", pm.ID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp deletePaymentMethodResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if !resp.Deleted {
		t.Error("expected deleted=true")
	}
	if resp.AffectedAgents != 2 {
		t.Errorf("expected affected_agents=2, got %d", resp.AffectedAgents)
	}
}

// authenticatedRequestWithBody creates a request with a JSON body and auth.
func authenticatedRequestWithBody(t *testing.T, method, path, userID string, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Copy auth header from authenticatedRequest.
	authReq := authenticatedRequest(t, method, path, userID)
	req.Header.Set("Authorization", authReq.Header.Get("Authorization"))
	return req
}
