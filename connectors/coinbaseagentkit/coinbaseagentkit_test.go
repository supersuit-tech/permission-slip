package coinbaseagentkit

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestValidateCredentials(t *testing.T) {
	c := New()
	ctx := context.Background()

	err := c.ValidateCredentials(ctx, connectors.NewCredentials(map[string]string{
		"api_key_id":     "kid",
		"api_key_secret": "secret",
		"wallet_secret":  "wallet",
	}))
	if err != nil {
		t.Fatalf("expected valid creds: %v", err)
	}

	err = c.ValidateCredentials(ctx, connectors.NewCredentials(map[string]string{
		"api_key_id": "kid",
	}))
	if err == nil {
		t.Fatal("expected error for missing secrets")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %T: %v", err, err)
	}
}

func TestParseSendNetwork(t *testing.T) {
	_, err := parseSendNetwork("base")
	if err != nil {
		t.Fatal(err)
	}
	_, err = parseSendNetwork("invalid-net")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseFaucetToken(t *testing.T) {
	_, err := parseFaucetToken("eth")
	if err != nil {
		t.Fatal(err)
	}
	_, err = parseFaucetToken("bitcoin")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManifest_Valid(t *testing.T) {
	c := New()
	mp, ok := any(c).(connectors.ManifestProvider)
	if !ok {
		t.Fatal("connector should implement ManifestProvider")
	}
	m := mp.Manifest()
	if m == nil {
		t.Fatal("Manifest() returned nil")
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("manifest validation failed: %v", err)
	}
	if m.ID != "coinbase_agentkit" {
		t.Errorf("manifest ID = %q, want coinbase_agentkit", m.ID)
	}
	if len(m.Actions) != 7 {
		t.Errorf("expected 7 actions, got %d", len(m.Actions))
	}
	if len(m.Templates) != 3 {
		t.Errorf("expected 3 templates, got %d", len(m.Templates))
	}
}
