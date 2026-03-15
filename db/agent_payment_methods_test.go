package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestAgentPaymentMethodCRUD(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "apm_"+uid[:8])
	agentID := testhelper.InsertAgent(t, tx, uid)

	// Create a payment method.
	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_apm_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "4242",
		ExpMonth:              12,
		ExpYear:               2028,
		IsDefault:             true,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	// Get should return nil initially.
	got, err := db.GetAgentPaymentMethod(ctx, tx, agentID)
	if err != nil {
		t.Fatalf("GetAgentPaymentMethod: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil before assignment")
	}

	// Assign.
	assigned, err := db.AssignAgentPaymentMethod(ctx, tx, agentID, pm.ID)
	if err != nil {
		t.Fatalf("AssignAgentPaymentMethod: %v", err)
	}
	if assigned.AgentID != agentID {
		t.Errorf("expected agent_id=%d, got %d", agentID, assigned.AgentID)
	}
	if assigned.PaymentMethodID != pm.ID {
		t.Errorf("expected payment_method_id=%s, got %s", pm.ID, assigned.PaymentMethodID)
	}

	// Get should return the assignment.
	got, err = db.GetAgentPaymentMethod(ctx, tx, agentID)
	if err != nil {
		t.Fatalf("GetAgentPaymentMethod after assign: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil after assignment")
	}
	if got.PaymentMethodID != pm.ID {
		t.Errorf("got payment_method_id=%s, want %s", got.PaymentMethodID, pm.ID)
	}

	// Count should be 1.
	count, err := db.CountAgentsByPaymentMethod(ctx, tx, pm.ID)
	if err != nil {
		t.Fatalf("CountAgentsByPaymentMethod: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count=1, got %d", count)
	}

	// Re-assign with a second payment method (upsert).
	pm2, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_apm2_" + uid[:8],
		Brand:                 "mastercard",
		Last4:                 "5555",
		ExpMonth:              6,
		ExpYear:               2027,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod 2: %v", err)
	}

	reassigned, err := db.AssignAgentPaymentMethod(ctx, tx, agentID, pm2.ID)
	if err != nil {
		t.Fatalf("AssignAgentPaymentMethod (upsert): %v", err)
	}
	if reassigned.PaymentMethodID != pm2.ID {
		t.Errorf("expected pm2 after upsert, got %s", reassigned.PaymentMethodID)
	}

	// Old PM count should be 0, new PM count should be 1.
	count, err = db.CountAgentsByPaymentMethod(ctx, tx, pm.ID)
	if err != nil {
		t.Fatalf("CountAgentsByPaymentMethod pm1: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count=0 for pm1 after reassign, got %d", count)
	}

	// Remove.
	deleted, err := db.RemoveAgentPaymentMethod(ctx, tx, agentID)
	if err != nil {
		t.Fatalf("RemoveAgentPaymentMethod: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	// Get should return nil after removal.
	got, err = db.GetAgentPaymentMethod(ctx, tx, agentID)
	if err != nil {
		t.Fatalf("GetAgentPaymentMethod after remove: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil after removal")
	}

	// Remove again should return false.
	deleted, err = db.RemoveAgentPaymentMethod(ctx, tx, agentID)
	if err != nil {
		t.Fatalf("RemoveAgentPaymentMethod again: %v", err)
	}
	if deleted {
		t.Error("expected deleted=false on second removal")
	}
}

func TestAgentPaymentMethodCascadeOnPMDelete(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "apmcas_"+uid[:8])
	agentID := testhelper.InsertAgent(t, tx, uid)

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_cas_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "1111",
		ExpMonth:              3,
		ExpYear:               2029,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	_, err = db.AssignAgentPaymentMethod(ctx, tx, agentID, pm.ID)
	if err != nil {
		t.Fatalf("AssignAgentPaymentMethod: %v", err)
	}

	// Delete the payment method — should cascade and remove the agent binding.
	_, err = db.DeletePaymentMethod(ctx, tx, uid, pm.ID)
	if err != nil {
		t.Fatalf("DeletePaymentMethod: %v", err)
	}

	got, err := db.GetAgentPaymentMethod(ctx, tx, agentID)
	if err != nil {
		t.Fatalf("GetAgentPaymentMethod after PM delete: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil after payment method cascade delete")
	}
}
