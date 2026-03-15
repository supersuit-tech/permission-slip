package stripe

import (
	"testing"
)

func TestFreeRequestAllowance(t *testing.T) {
	if FreeRequestAllowance() != 250 {
		t.Errorf("expected FreeRequestAllowance()=250 (from config/plans.json free plan), got %d", FreeRequestAllowance())
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

func TestOverageCostCents(t *testing.T) {
	tests := []struct {
		overage  int
		expected int
	}{
		{0, 0},
		{-1, 0},
		{1, 1},   // ceil(0.5) = 1 cent
		{2, 1},   // ceil(1.0) = 1 cent
		{10, 5},  // ceil(5.0) = 5 cents
		{50, 25}, // ceil(25.0) = 25 cents
		{99, 50}, // ceil(49.5) = 50 cents
		{100, 50},
		{1000, 500},
	}
	for _, tt := range tests {
		got := OverageCostCents(tt.overage)
		if got != tt.expected {
			t.Errorf("OverageCostCents(%d) = %d, want %d", tt.overage, got, tt.expected)
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

func TestGetRequestPrice_NilWhenNotFetched(t *testing.T) {
	client := New(Config{})
	if rp := client.GetRequestPrice(); rp != nil {
		t.Errorf("expected nil request price before fetch, got %+v", rp)
	}
}

func TestRequestPriceDisplay_FallbackToPlansJSON(t *testing.T) {
	client := New(Config{})
	display := client.RequestPriceDisplay()
	if display != "$0.005" {
		t.Errorf("expected fallback $0.005, got %q", display)
	}
}

func TestRequestPriceDisplay_PackageLevel(t *testing.T) {
	// Package-level function should work even without explicit client setup
	// (New() sets cachedClient).
	_ = New(Config{})
	display := RequestPriceDisplay()
	if display != "$0.005" {
		t.Errorf("expected package-level fallback $0.005, got %q", display)
	}
}
