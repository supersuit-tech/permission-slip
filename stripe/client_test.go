package stripe

import (
	"testing"
)

func TestFreeRequestAllowance(t *testing.T) {
	if FreeRequestAllowance != 1000 {
		t.Errorf("expected FreeRequestAllowance=1000, got %d", FreeRequestAllowance)
	}
}

func TestSMSRates_AllRegionsExist(t *testing.T) {
	expectedRegions := []string{"us_ca", "uk_eu", "international"}
	for _, region := range expectedRegions {
		rate, ok := SMSRates[region]
		if !ok {
			t.Errorf("expected SMS rate for region %q", region)
			continue
		}
		if rate.UnitAmount <= 0 {
			t.Errorf("expected positive unit amount for region %q, got %d", region, rate.UnitAmount)
		}
		if rate.Description == "" {
			t.Errorf("expected non-empty description for region %q", region)
		}
	}
}

func TestSMSRates_PricingCorrect(t *testing.T) {
	tests := []struct {
		region   string
		expected int64
	}{
		{"us_ca", 1},         // $0.01
		{"uk_eu", 4},         // $0.04
		{"international", 5}, // $0.05
	}
	for _, tt := range tests {
		rate := SMSRates[tt.region]
		if rate.UnitAmount != tt.expected {
			t.Errorf("region %q: expected UnitAmount=%d cents, got %d", tt.region, tt.expected, rate.UnitAmount)
		}
	}
}

func TestNew_SetsAPIKey(t *testing.T) {
	// Ensure New doesn't panic with empty config.
	client := New(Config{})
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestWebhookSecret(t *testing.T) {
	secret := "whsec_test_secret"
	client := New(Config{WebhookSecret: secret})
	if client.WebhookSecret() != secret {
		t.Errorf("expected webhook secret %q, got %q", secret, client.WebhookSecret())
	}
}
