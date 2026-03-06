package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// PaymentMethod represents a stored payment method (credit/debit card).
// Raw card data is never stored — only the Stripe payment method token
// and display metadata (last4, brand, expiry).
type PaymentMethod struct {
	ID                    string     `json:"id"`
	UserID                string     `json:"user_id"`
	StripePaymentMethodID string     `json:"stripe_payment_method_id"`
	Label                 string     `json:"label"`
	Brand                 string     `json:"brand"`
	Last4                 string     `json:"last4"`
	ExpMonth              int        `json:"exp_month"`
	ExpYear               int        `json:"exp_year"`
	BillingAddressLine1   string     `json:"billing_address_line1"`
	BillingAddressLine2   string     `json:"billing_address_line2"`
	BillingAddressCity    string     `json:"billing_address_city"`
	BillingAddressState   string     `json:"billing_address_state"`
	BillingAddressPostal  string     `json:"billing_address_postal"`
	BillingAddressCountry string     `json:"billing_address_country"`
	IsDefault             bool       `json:"is_default"`
	PerTransactionLimit   *int       `json:"per_transaction_limit"`
	MonthlyLimit          *int       `json:"monthly_limit"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

const paymentMethodColumns = `id, user_id, stripe_payment_method_id, label, brand, last4,
	exp_month, exp_year,
	billing_address_line1, billing_address_line2, billing_address_city,
	billing_address_state, billing_address_postal, billing_address_country,
	is_default, per_transaction_limit, monthly_limit, created_at, updated_at`

func scanPaymentMethod(row pgx.Row) (*PaymentMethod, error) {
	var pm PaymentMethod
	err := row.Scan(
		&pm.ID, &pm.UserID, &pm.StripePaymentMethodID,
		&pm.Label, &pm.Brand, &pm.Last4,
		&pm.ExpMonth, &pm.ExpYear,
		&pm.BillingAddressLine1, &pm.BillingAddressLine2, &pm.BillingAddressCity,
		&pm.BillingAddressState, &pm.BillingAddressPostal, &pm.BillingAddressCountry,
		&pm.IsDefault, &pm.PerTransactionLimit, &pm.MonthlyLimit,
		&pm.CreatedAt, &pm.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

// CreatePaymentMethod inserts a new payment method for the user.
func CreatePaymentMethod(ctx context.Context, db DBTX, pm *PaymentMethod) (*PaymentMethod, error) {
	return scanPaymentMethod(db.QueryRow(ctx,
		`INSERT INTO payment_methods (
			user_id, stripe_payment_method_id, label, brand, last4,
			exp_month, exp_year,
			billing_address_line1, billing_address_line2, billing_address_city,
			billing_address_state, billing_address_postal, billing_address_country,
			is_default, per_transaction_limit, monthly_limit
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING `+paymentMethodColumns,
		pm.UserID, pm.StripePaymentMethodID, pm.Label, pm.Brand, pm.Last4,
		pm.ExpMonth, pm.ExpYear,
		pm.BillingAddressLine1, pm.BillingAddressLine2, pm.BillingAddressCity,
		pm.BillingAddressState, pm.BillingAddressPostal, pm.BillingAddressCountry,
		pm.IsDefault, pm.PerTransactionLimit, pm.MonthlyLimit,
	))
}

// ListPaymentMethodsByUser returns all payment methods for a user, ordered by
// default status (default first) then creation date.
func ListPaymentMethodsByUser(ctx context.Context, db DBTX, userID string) ([]PaymentMethod, error) {
	rows, err := db.Query(ctx,
		`SELECT `+paymentMethodColumns+` FROM payment_methods
		 WHERE user_id = $1
		 ORDER BY is_default DESC, created_at ASC`,
		userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var methods []PaymentMethod
	for rows.Next() {
		var pm PaymentMethod
		if err := rows.Scan(
			&pm.ID, &pm.UserID, &pm.StripePaymentMethodID,
			&pm.Label, &pm.Brand, &pm.Last4,
			&pm.ExpMonth, &pm.ExpYear,
			&pm.BillingAddressLine1, &pm.BillingAddressLine2, &pm.BillingAddressCity,
			&pm.BillingAddressState, &pm.BillingAddressPostal, &pm.BillingAddressCountry,
			&pm.IsDefault, &pm.PerTransactionLimit, &pm.MonthlyLimit,
			&pm.CreatedAt, &pm.UpdatedAt,
		); err != nil {
			return nil, err
		}
		methods = append(methods, pm)
	}
	return methods, rows.Err()
}

// GetPaymentMethodByID returns a single payment method by ID, scoped to the user.
func GetPaymentMethodByID(ctx context.Context, db DBTX, userID, paymentMethodID string) (*PaymentMethod, error) {
	pm, err := scanPaymentMethod(db.QueryRow(ctx,
		`SELECT `+paymentMethodColumns+` FROM payment_methods WHERE id = $1 AND user_id = $2`,
		paymentMethodID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return pm, err
}

// UpdatePaymentMethodFields updates mutable fields (label, limits, default status).
type UpdatePaymentMethodParams struct {
	Label               *string
	IsDefault           *bool
	PerTransactionLimit *int  // use pointer-to-nil pattern: nil = don't change, non-nil = set (value can be 0 to clear)
	MonthlyLimit        *int
	ClearPerTxLimit     bool // when true, sets per_transaction_limit to NULL
	ClearMonthlyLimit   bool // when true, sets monthly_limit to NULL
}

// UpdatePaymentMethod updates a payment method's mutable fields.
func UpdatePaymentMethod(ctx context.Context, db DBTX, userID, paymentMethodID string, params UpdatePaymentMethodParams) (*PaymentMethod, error) {
	pm, err := scanPaymentMethod(db.QueryRow(ctx,
		`UPDATE payment_methods SET
			label = COALESCE($3, label),
			is_default = COALESCE($4, is_default),
			per_transaction_limit = CASE
				WHEN $7 THEN NULL
				WHEN $5::int IS NOT NULL THEN $5
				ELSE per_transaction_limit
			END,
			monthly_limit = CASE
				WHEN $8 THEN NULL
				WHEN $6::int IS NOT NULL THEN $6
				ELSE monthly_limit
			END,
			updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+paymentMethodColumns,
		paymentMethodID, userID,
		params.Label, params.IsDefault,
		params.PerTransactionLimit, params.MonthlyLimit,
		params.ClearPerTxLimit, params.ClearMonthlyLimit,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return pm, err
}

// DeletePaymentMethod removes a payment method by ID, scoped to the user.
// Returns true if a row was deleted.
func DeletePaymentMethod(ctx context.Context, db DBTX, userID, paymentMethodID string) (bool, error) {
	tag, err := db.Exec(ctx,
		`DELETE FROM payment_methods WHERE id = $1 AND user_id = $2`,
		paymentMethodID, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ClearDefaultPaymentMethod removes the default flag from all payment methods
// for a user. Used before setting a new default.
func ClearDefaultPaymentMethod(ctx context.Context, db DBTX, userID string) error {
	_, err := db.Exec(ctx,
		`UPDATE payment_methods SET is_default = false, updated_at = now() WHERE user_id = $1 AND is_default = true`,
		userID)
	return err
}

// CountPaymentMethodsByUser returns the number of payment methods for a user.
func CountPaymentMethodsByUser(ctx context.Context, db DBTX, userID string) (int, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM payment_methods WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}

// PaymentMethodTransaction records a charge against a stored payment method.
type PaymentMethodTransaction struct {
	ID              string    `json:"id"`
	PaymentMethodID string    `json:"payment_method_id"`
	UserID          string    `json:"user_id"`
	ConnectorID     string    `json:"connector_id"`
	ActionType      string    `json:"action_type"`
	AmountCents     int       `json:"amount_cents"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreatePaymentMethodTransaction records a transaction against a payment method.
func CreatePaymentMethodTransaction(ctx context.Context, db DBTX, tx *PaymentMethodTransaction) (*PaymentMethodTransaction, error) {
	var t PaymentMethodTransaction
	err := db.QueryRow(ctx,
		`INSERT INTO payment_method_transactions (payment_method_id, user_id, connector_id, action_type, amount_cents, description)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, payment_method_id, user_id, connector_id, action_type, amount_cents, description, created_at`,
		tx.PaymentMethodID, tx.UserID, tx.ConnectorID, tx.ActionType, tx.AmountCents, tx.Description,
	).Scan(&t.ID, &t.PaymentMethodID, &t.UserID, &t.ConnectorID, &t.ActionType, &t.AmountCents, &t.Description, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetMonthlySpend returns the total amount_cents spent using a payment method
// in the last 30 days (rolling window).
func GetMonthlySpend(ctx context.Context, db DBTX, paymentMethodID string) (int, error) {
	var total int
	err := db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_cents), 0) FROM payment_method_transactions
		 WHERE payment_method_id = $1 AND created_at >= now() - interval '30 days'`,
		paymentMethodID).Scan(&total)
	return total, err
}
