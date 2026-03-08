package stripe

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ---------------------------------------------------------------------------
// Connector interface compliance
// ---------------------------------------------------------------------------

// Compile-time interface checks (matches GitHub/Slack connector pattern).
var _ connectors.Connector = (*StripeConnector)(nil)
var _ connectors.ManifestProvider = (*StripeConnector)(nil)

func TestManifest_Valid(t *testing.T) {
	t.Parallel()

	conn := New()
	m := conn.Manifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("Manifest().Validate() error: %v", err)
	}
	if m.ID != "stripe" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "stripe")
	}
	if len(m.Actions) != 19 {
		t.Errorf("Manifest().Actions has %d entries, want 19", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 2 {
		t.Errorf("Manifest().RequiredCredentials has %d entries, want 2", len(m.RequiredCredentials))
	}
	// OAuth should be first (default/primary auth method).
	if m.RequiredCredentials[0].AuthType != "oauth2" {
		t.Errorf("RequiredCredentials[0].AuthType = %q, want %q", m.RequiredCredentials[0].AuthType, "oauth2")
	}
	if m.RequiredCredentials[0].OAuthProvider != "stripe" {
		t.Errorf("RequiredCredentials[0].OAuthProvider = %q, want %q", m.RequiredCredentials[0].OAuthProvider, "stripe")
	}
	// API key should be second (alternative auth method).
	if m.RequiredCredentials[1].AuthType != "api_key" {
		t.Errorf("RequiredCredentials[1].AuthType = %q, want %q", m.RequiredCredentials[1].AuthType, "api_key")
	}
	if len(m.Templates) != 26 {
		t.Errorf("Manifest().Templates has %d entries, want 26", len(m.Templates))
	}
}

func TestManifest_ActionsMatchRegistered(t *testing.T) {
	t.Parallel()

	conn := New()
	m := conn.Manifest()
	actions := conn.Actions()

	for _, ma := range m.Actions {
		if _, ok := actions[ma.ActionType]; !ok {
			t.Errorf("Manifest action %q not found in Actions() map", ma.ActionType)
		}
	}
	for actionType := range actions {
		found := false
		for _, ma := range m.Actions {
			if ma.ActionType == actionType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Actions() has %q but it's not in the Manifest", actionType)
		}
	}
}

func TestID(t *testing.T) {
	t.Parallel()

	conn := New()
	if conn.ID() != "stripe" {
		t.Errorf("ID() = %q, want %q", conn.ID(), "stripe")
	}
}

func TestActions_ReturnsMap(t *testing.T) {
	t.Parallel()

	conn := New()
	actions := conn.Actions()
	if actions == nil {
		t.Fatal("Actions() returned nil")
	}
	expectedActions := []string{
		"stripe.create_customer",
		"stripe.create_invoice",
		"stripe.issue_refund",
		"stripe.list_subscriptions",
		"stripe.create_payment_link",
		"stripe.get_balance",
		"stripe.create_subscription",
		"stripe.cancel_subscription",
		"stripe.create_coupon",
		"stripe.create_promotion_code",
		"stripe.initiate_payout",
		"stripe.create_checkout_session",
		"stripe.create_product",
		"stripe.create_price",
		"stripe.update_subscription",
		"stripe.list_customers",
		"stripe.get_customer",
		"stripe.list_invoices",
		"stripe.list_charges",
	}
	if len(actions) != len(expectedActions) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(expectedActions))
	}
	for _, name := range expectedActions {
		if _, ok := actions[name]; !ok {
			t.Errorf("Actions() missing %q", name)
		}
	}
}
