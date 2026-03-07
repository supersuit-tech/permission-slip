package testhelper

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// PaymentMethodOpts holds optional fields for InsertPaymentMethod.
type PaymentMethodOpts struct {
	Label               string
	Brand               string
	Last4               string
	PerTransactionLimit *int
	MonthlyLimit        *int
	IsDefault           bool
}

// InsertPaymentMethod inserts a payment method for the given user with a generated
// Stripe payment method ID. Returns the payment method ID.
func InsertPaymentMethod(t *testing.T, d db.DBTX, userID string, opts PaymentMethodOpts) string {
	t.Helper()

	stripeID := "pm_test_" + GenerateID(t, "")
	if opts.Brand == "" {
		opts.Brand = "visa"
	}
	if opts.Last4 == "" {
		opts.Last4 = "4242"
	}

	pm := &db.PaymentMethod{
		UserID:                userID,
		StripePaymentMethodID: stripeID,
		Label:                 opts.Label,
		Brand:                 opts.Brand,
		Last4:                 opts.Last4,
		ExpMonth:              12,
		ExpYear:               2030,
		IsDefault:             opts.IsDefault,
		PerTransactionLimit:   opts.PerTransactionLimit,
		MonthlyLimit:          opts.MonthlyLimit,
	}

	created, err := db.CreatePaymentMethod(t.Context(), d, pm)
	if err != nil {
		t.Fatalf("InsertPaymentMethod: %v", err)
	}
	return created.ID
}
