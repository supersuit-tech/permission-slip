package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func TestConnectorIDFromActionType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  *string
	}{
		{"github.create_issue", strPtr("github")},
		{"slack.send_message", strPtr("slack")},
		{"email.send", strPtr("email")},
		{"connector.deeply.nested.action", strPtr("connector")},
		{"malformed_type", nil},
		{".missing_prefix", nil},
		{"", nil},
		{"trailing.", strPtr("trailing")},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := connectorIDFromActionType(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("connectorIDFromActionType(%q) = %q, want nil", tt.input, *got)
				}
			} else {
				if got == nil {
					t.Errorf("connectorIDFromActionType(%q) = nil, want %q", tt.input, *tt.want)
				} else if *got != *tt.want {
					t.Errorf("connectorIDFromActionType(%q) = %q, want %q", tt.input, *got, *tt.want)
				}
			}
		})
	}
}

func TestActionTypeFromJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"valid", []byte(`{"type":"email.send","version":"1"}`), "email.send"},
		{"type only", []byte(`{"type":"github.create_issue"}`), "github.create_issue"},
		{"empty type", []byte(`{"type":""}`), ""},
		{"no type field", []byte(`{"action":"something"}`), ""},
		{"empty json", []byte(`{}`), ""},
		{"nil input", nil, ""},
		{"empty bytes", []byte{}, ""},
		{"invalid json", []byte(`not json`), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := actionTypeFromJSON(tt.input); got != tt.want {
				t.Errorf("actionTypeFromJSON(%s) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRedactActionToType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantNil bool
	}{
		{"valid", []byte(`{"type":"email.send","parameters":{"to":"alice@example.com"}}`), `{"type":"email.send"}`, false},
		{"type only", []byte(`{"type":"github.create_issue"}`), `{"type":"github.create_issue"}`, false},
		{"empty type", []byte(`{"type":""}`), "", true},
		{"no type", []byte(`{"action":"something"}`), "", true},
		{"nil", nil, "", true},
		{"empty", []byte{}, "", true},
		{"invalid json", []byte(`not json`), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactActionToType(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("redactActionToType(%s) = %s, want nil", tt.input, got)
				}
			} else {
				if string(got) != tt.want {
					t.Errorf("redactActionToType(%s) = %s, want %s", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestResolveExecResult(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		status, errMsg := resolveExecResult(nil)
		if status != db.ExecStatusSuccess {
			t.Errorf("status = %q, want %q", status, db.ExecStatusSuccess)
		}
		if errMsg != nil {
			t.Errorf("errMsg = %q, want nil", *errMsg)
		}
	})

	t.Run("failure", func(t *testing.T) {
		status, errMsg := resolveExecResult(errors.New("connection refused"))
		if status != db.ExecStatusFailure {
			t.Errorf("status = %q, want %q", status, db.ExecStatusFailure)
		}
		if errMsg == nil {
			t.Fatal("errMsg = nil, want non-nil")
		}
		if *errMsg != "connection refused" {
			t.Errorf("errMsg = %q, want %q", *errMsg, "connection refused")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		status, errMsg := resolveExecResult(context.DeadlineExceeded)
		if status != db.ExecStatusTimeout {
			t.Errorf("status = %q, want %q", status, db.ExecStatusTimeout)
		}
		if errMsg == nil {
			t.Fatal("errMsg = nil, want non-nil")
		}
	})

	t.Run("wrapped timeout", func(t *testing.T) {
		wrappedErr := fmt.Errorf("connector execution: %w", context.DeadlineExceeded)
		status, errMsg := resolveExecResult(wrappedErr)
		if status != db.ExecStatusTimeout {
			t.Errorf("status = %q, want %q", status, db.ExecStatusTimeout)
		}
		if errMsg == nil {
			t.Fatal("errMsg = nil, want non-nil")
		}
	})

	t.Run("truncation", func(t *testing.T) {
		longErr := errors.New(strings.Repeat("x", 1000))
		status, errMsg := resolveExecResult(longErr)
		if status != db.ExecStatusFailure {
			t.Errorf("status = %q, want %q", status, db.ExecStatusFailure)
		}
		if errMsg == nil {
			t.Fatal("errMsg = nil, want non-nil")
		}
		if len(*errMsg) != 512 {
			t.Errorf("len(errMsg) = %d, want 512", len(*errMsg))
		}
	})
}

func TestRecordPaymentMethodUsage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("CreatesTransactionAndAuditEvent", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		// Create a payment method so the FK on payment_method_transactions is satisfied.
		pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
			UserID:                uid,
			StripePaymentMethodID: "pm_test_" + uid[:8],
			Label:                 "Test Card",
			Brand:                 "visa",
			Last4:                 "4242",
			ExpMonth:              12,
			ExpYear:               2027,
			IsDefault:             true,
		})
		if err != nil {
			t.Fatalf("CreatePaymentMethod: %v", err)
		}

		RecordPaymentMethodUsage(ctx, tx, PaymentChargeParams{
			UserID:          uid,
			AgentID:         agentID,
			AgentMeta:       []byte(`{"name":"travel-agent"}`),
			PaymentMethodID: pm.ID,
			Brand:           "visa",
			Last4:           "4242",
			ConnectorID:     "expedia",
			ActionType:      "expedia.create_booking",
			AmountCents:     15000,
			Currency:        "usd",
			Description:     "Hotel booking",
		})

		// Verify the payment method transaction was created.
		spend, err := db.GetMonthlySpend(ctx, tx, pm.ID)
		if err != nil {
			t.Fatalf("GetMonthlySpend: %v", err)
		}
		if spend != 15000 {
			t.Errorf("expected 15000 monthly spend, got %d", spend)
		}

		// Verify the audit event was created.
		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, &db.AuditEventFilter{
			EventTypes: []db.AuditEventType{db.AuditEventPaymentMethodCharged},
		}, 0)
		if err != nil {
			t.Fatalf("ListAuditEvents: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 audit event, got %d", len(page.Events))
		}

		e := page.Events[0]
		if e.EventType != db.AuditEventPaymentMethodCharged {
			t.Errorf("expected payment_method.charged, got %s", e.EventType)
		}
		if e.Outcome != "charged" {
			t.Errorf("expected outcome charged, got %q", e.Outcome)
		}
		if e.ConnectorID == nil || *e.ConnectorID != "expedia" {
			t.Errorf("expected connector_id=expedia, got %v", e.ConnectorID)
		}

		// Verify the action contains safe metadata and no raw card details.
		var action map[string]json.RawMessage
		if err := json.Unmarshal(e.Action, &action); err != nil {
			t.Fatalf("failed to parse action JSON: %v", err)
		}

		requiredFields := []string{"type", "payment_method_id", "brand", "last4", "amount_cents", "currency", "description"}
		for _, field := range requiredFields {
			if _, ok := action[field]; !ok {
				t.Errorf("action missing field %q", field)
			}
		}

		// Verify the amount is correct in the action JSON.
		var amountCents int
		if err := json.Unmarshal(action["amount_cents"], &amountCents); err != nil {
			t.Fatalf("failed to parse amount_cents: %v", err)
		}
		if amountCents != 15000 {
			t.Errorf("expected amount_cents=15000, got %d", amountCents)
		}

		// Verify the description is included.
		var desc string
		if err := json.Unmarshal(action["description"], &desc); err != nil {
			t.Fatalf("failed to parse description: %v", err)
		}
		if desc != "Hotel booking" {
			t.Errorf("expected description=%q, got %q", "Hotel booking", desc)
		}
	})

	t.Run("DefaultsCurrencyToUSD", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
			UserID:                uid,
			StripePaymentMethodID: "pm_test2_" + uid[:8],
			Label:                 "Test Card",
			Brand:                 "mastercard",
			Last4:                 "5678",
			ExpMonth:              6,
			ExpYear:               2028,
		})
		if err != nil {
			t.Fatalf("CreatePaymentMethod: %v", err)
		}

		RecordPaymentMethodUsage(ctx, tx, PaymentChargeParams{
			UserID:          uid,
			AgentID:         agentID,
			AgentMeta:       []byte(`{"name":"test"}`),
			PaymentMethodID: pm.ID,
			Brand:           "mastercard",
			Last4:           "5678",
			ConnectorID:     "amazon",
			ActionType:      "amazon.purchase",
			AmountCents:     999,
			Currency:        "", // should default to "usd"
			Description:     "Purchase",
		})

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("ListAuditEvents: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}

		var action map[string]json.RawMessage
		if err := json.Unmarshal(page.Events[0].Action, &action); err != nil {
			t.Fatalf("failed to parse action JSON: %v", err)
		}

		var currency string
		if err := json.Unmarshal(action["currency"], &currency); err != nil {
			t.Fatalf("failed to parse currency: %v", err)
		}
		if currency != "usd" {
			t.Errorf("expected currency=usd, got %q", currency)
		}
	})

	t.Run("NeverLogsRawCardDetails", func(t *testing.T) {
		t.Parallel()
		tx := testhelper.SetupTestDB(t)
		uid := testhelper.GenerateUID(t)
		agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

		pm, err := db.CreatePaymentMethod(ctx, tx, &db.PaymentMethod{
			UserID:                uid,
			StripePaymentMethodID: "pm_test3_" + uid[:8],
			Label:                 "Test Card",
			Brand:                 "amex",
			Last4:                 "0005",
			ExpMonth:              3,
			ExpYear:               2029,
		})
		if err != nil {
			t.Fatalf("CreatePaymentMethod: %v", err)
		}

		RecordPaymentMethodUsage(ctx, tx, PaymentChargeParams{
			UserID:          uid,
			AgentID:         agentID,
			AgentMeta:       []byte(`{"name":"test"}`),
			PaymentMethodID: pm.ID,
			Brand:           "amex",
			Last4:           "0005",
			ConnectorID:     "doordash",
			ActionType:      "doordash.place_order",
			AmountCents:     2500,
			Currency:        "usd",
			Description:     "Food delivery",
		})

		page, err := db.ListAuditEvents(ctx, tx, uid, 20, nil, nil, 0)
		if err != nil {
			t.Fatalf("ListAuditEvents: %v", err)
		}
		if len(page.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(page.Events))
		}

		actionStr := string(page.Events[0].Action)
		// Ensure no full card number patterns appear.
		sensitivePatterns := []string{"card_number", "cvv", "full_number", "expiry"}
		for _, pattern := range sensitivePatterns {
			if strings.Contains(actionStr, pattern) {
				t.Errorf("action JSON must not contain %q, got: %s", pattern, actionStr)
			}
		}

		// Verify only safe fields: last4 and brand are present, not full card info.
		var action map[string]json.RawMessage
		if err := json.Unmarshal(page.Events[0].Action, &action); err != nil {
			t.Fatalf("failed to parse action JSON: %v", err)
		}
		if _, ok := action["last4"]; !ok {
			t.Error("expected last4 in action")
		}
		if _, ok := action["brand"]; !ok {
			t.Error("expected brand in action")
		}
	})
}

func strPtr(s string) *string { return &s }
