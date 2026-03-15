package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// AgentPaymentMethod represents the binding between an agent and a payment method.
type AgentPaymentMethod struct {
	AgentID         int64     `json:"agent_id"`
	PaymentMethodID string    `json:"payment_method_id"`
	CreatedAt       time.Time `json:"created_at"`
}

// GetAgentPaymentMethod returns the payment method assignment for an agent, or nil if none.
func GetAgentPaymentMethod(ctx context.Context, db DBTX, agentID int64) (*AgentPaymentMethod, error) {
	var apm AgentPaymentMethod
	err := db.QueryRow(ctx,
		`SELECT agent_id, payment_method_id, created_at
		 FROM agent_payment_methods WHERE agent_id = $1`,
		agentID).Scan(&apm.AgentID, &apm.PaymentMethodID, &apm.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &apm, nil
}

// AssignAgentPaymentMethod upserts the payment method for an agent.
func AssignAgentPaymentMethod(ctx context.Context, db DBTX, agentID int64, paymentMethodID string) (*AgentPaymentMethod, error) {
	var apm AgentPaymentMethod
	err := db.QueryRow(ctx,
		`INSERT INTO agent_payment_methods (agent_id, payment_method_id)
		 VALUES ($1, $2)
		 ON CONFLICT (agent_id) DO UPDATE SET payment_method_id = EXCLUDED.payment_method_id, created_at = now()
		 RETURNING agent_id, payment_method_id, created_at`,
		agentID, paymentMethodID).Scan(&apm.AgentID, &apm.PaymentMethodID, &apm.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &apm, nil
}

// RemoveAgentPaymentMethod deletes the payment method assignment for an agent.
// Returns true if a row was deleted.
func RemoveAgentPaymentMethod(ctx context.Context, db DBTX, agentID int64) (bool, error) {
	tag, err := db.Exec(ctx,
		`DELETE FROM agent_payment_methods WHERE agent_id = $1`,
		agentID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// CountAgentsByPaymentMethod returns the number of agents using a given payment method.
func CountAgentsByPaymentMethod(ctx context.Context, db DBTX, paymentMethodID string) (int, error) {
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM agent_payment_methods WHERE payment_method_id = $1`,
		paymentMethodID).Scan(&count)
	return count, err
}
