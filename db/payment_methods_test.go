package db_test

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestPaymentMethodCRUD(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmtest_"+uid[:8])

	// List should be empty initially.
	methods, err := db.ListPaymentMethodsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("ListPaymentMethodsByUser: %v", err)
	}
	if len(methods) != 0 {
		t.Fatalf("expected 0 payment methods, got %d", len(methods))
	}

	// Create a payment method.
	pm := &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_test_" + uid[:8],
		Label:                 "Personal Visa",
		Brand:                 "visa",
		Last4:                 "4242",
		ExpMonth:              12,
		ExpYear:               2027,
		IsDefault:             true,
	}
	created, err := db.CreatePaymentMethod(ctx, tx, pm)
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if created.Brand != "visa" {
		t.Errorf("expected brand=visa, got %q", created.Brand)
	}
	if created.Last4 != "4242" {
		t.Errorf("expected last4=4242, got %q", created.Last4)
	}
	if !created.IsDefault {
		t.Error("expected is_default=true")
	}

	// Get by ID.
	fetched, err := db.GetPaymentMethodByID(ctx, tx, uid, created.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodByID: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected payment method, got nil")
	}
	if fetched.Label != "Personal Visa" {
		t.Errorf("expected label='Personal Visa', got %q", fetched.Label)
	}

	// List should have 1 method.
	methods, err = db.ListPaymentMethodsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("ListPaymentMethodsByUser: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 payment method, got %d", len(methods))
	}

	// Update label and limits.
	newLabel := "Work Card"
	perTxLimit := 5000
	updated, err := db.UpdatePaymentMethod(ctx, tx, uid, created.ID, db.UpdatePaymentMethodParams{
		Label:               &newLabel,
		PerTransactionLimit: &perTxLimit,
	})
	if err != nil {
		t.Fatalf("UpdatePaymentMethod: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated payment method, got nil")
	}
	if updated.Label != "Work Card" {
		t.Errorf("expected label='Work Card', got %q", updated.Label)
	}
	if updated.PerTransactionLimit == nil || *updated.PerTransactionLimit != 5000 {
		t.Errorf("expected per_transaction_limit=5000, got %v", updated.PerTransactionLimit)
	}

	// Clear the per-transaction limit.
	cleared, err := db.UpdatePaymentMethod(ctx, tx, uid, created.ID, db.UpdatePaymentMethodParams{
		ClearPerTxLimit: true,
	})
	if err != nil {
		t.Fatalf("UpdatePaymentMethod (clear limit): %v", err)
	}
	if cleared.PerTransactionLimit != nil {
		t.Errorf("expected per_transaction_limit=nil after clear, got %v", cleared.PerTransactionLimit)
	}

	// Delete.
	deleted, err := db.DeletePaymentMethod(ctx, tx, uid, created.ID)
	if err != nil {
		t.Fatalf("DeletePaymentMethod: %v", err)
	}
	if !deleted {
		t.Error("expected delete to return true")
	}

	// Verify deleted.
	fetched, err = db.GetPaymentMethodByID(ctx, tx, uid, created.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodByID after delete: %v", err)
	}
	if fetched != nil {
		t.Error("expected nil after delete")
	}
}

func TestPaymentMethodAccessControl(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()

	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "pmowner_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "pmother_"+uid2[:8])

	// User1 creates a payment method.
	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid1,
		StripePaymentMethodID: "pm_access_" + uid1[:8],
		Brand:                 "mastercard",
		Last4:                 "5555",
		ExpMonth:              6,
		ExpYear:               2028,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	// User2 should not be able to see User1's payment method.
	fetched, err := db.GetPaymentMethodByID(ctx, tx, uid2, pm.ID)
	if err != nil {
		t.Fatalf("GetPaymentMethodByID (wrong user): %v", err)
	}
	if fetched != nil {
		t.Error("expected nil for another user's payment method")
	}

	// User2 should not be able to delete User1's payment method.
	deleted, err := db.DeletePaymentMethod(ctx, tx, uid2, pm.ID)
	if err != nil {
		t.Fatalf("DeletePaymentMethod (wrong user): %v", err)
	}
	if deleted {
		t.Error("expected delete to return false for another user's payment method")
	}
}

func TestPaymentMethodTransactions(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmtx_"+uid[:8])

	pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_tx_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "1234",
		ExpMonth:              3,
		ExpYear:               2029,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	// Monthly spend should be 0 initially.
	spend, err := db.GetMonthlySpend(ctx, tx, pm.ID)
	if err != nil {
		t.Fatalf("GetMonthlySpend: %v", err)
	}
	if spend != 0 {
		t.Errorf("expected 0 monthly spend, got %d", spend)
	}

	// Record some transactions.
	_, err = db.CreatePaymentMethodTransaction(ctx, tx, &db.PaymentMethodTransaction{
		PaymentMethodID: pm.ID,
		UserID:          uid,
		ConnectorID:     "expedia",
		ActionType:      "create_booking",
		AmountCents:     15000,
		Description:     "Hotel booking",
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethodTransaction: %v", err)
	}

	_, err = db.CreatePaymentMethodTransaction(ctx, tx, &db.PaymentMethodTransaction{
		PaymentMethodID: pm.ID,
		UserID:          uid,
		ConnectorID:     "expedia",
		ActionType:      "create_booking",
		AmountCents:     8500,
		Description:     "Flight booking",
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethodTransaction: %v", err)
	}

	// Monthly spend should be 23500.
	spend, err = db.GetMonthlySpend(ctx, tx, pm.ID)
	if err != nil {
		t.Fatalf("GetMonthlySpend: %v", err)
	}
	if spend != 23500 {
		t.Errorf("expected 23500 monthly spend, got %d", spend)
	}
}

func TestPaymentMethodCount(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	ctx := context.Background()
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "pmcount_"+uid[:8])

	count, err := db.CountPaymentMethodsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountPaymentMethodsByUser: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	_, err = db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_count1_" + uid[:8],
		Brand:                 "visa",
		Last4:                 "1111",
		ExpMonth:              1,
		ExpYear:               2030,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	_, err = db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
		UserID:                uid,
		StripePaymentMethodID: "pm_count2_" + uid[:8],
		Brand:                 "mastercard",
		Last4:                 "2222",
		ExpMonth:              2,
		ExpYear:               2030,
	})
	if err != nil {
		t.Fatalf("CreatePaymentMethod: %v", err)
	}

	count, err = db.CountPaymentMethodsByUser(ctx, tx, uid)
	if err != nil {
		t.Fatalf("CountPaymentMethodsByUser: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}
