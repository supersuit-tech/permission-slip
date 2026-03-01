package main

import (
	"os"
	"testing"
)

// setEnvForTest sets multiple env vars for the duration of a test and restores
// their previous values (or unsets them) when the test completes.
func setEnvForTest(t *testing.T, vars map[string]string) {
	t.Helper()
	originals := make(map[string]*string, len(vars))
	for k := range vars {
		if v, ok := os.LookupEnv(k); ok {
			v := v // capture
			originals[k] = &v
		} else {
			originals[k] = nil
		}
	}
	t.Cleanup(func() {
		for k, orig := range originals {
			if orig == nil {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, *orig)
			}
		}
	})
	for k, v := range vars {
		os.Setenv(k, v)
	}
}

func TestValidateConfig_DevelopmentModeSkipsErrors(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                "development",
		"DATABASE_URL":        "",
		"SUPABASE_URL":        "",
		"SUPABASE_JWT_SECRET": "",
		"SUPABASE_JWKS_URL":   "",
		"INVITE_HMAC_KEY":     "",
		"BASE_URL":            "",
	})

	errs, warnings := validateConfig()
	if len(errs) != 0 {
		t.Errorf("expected no errors in dev mode, got %d: %v", len(errs), errs)
	}
	// Warnings are still emitted in dev mode (missing INVITE_HMAC_KEY, BASE_URL).
	if len(warnings) == 0 {
		t.Error("expected warnings even in dev mode")
	}
}

func TestValidateConfig_DevelopmentModeNoWarningsWhenConfigured(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                       "development",
		"DATABASE_URL":               "",
		"SUPABASE_URL":               "",
		"SUPABASE_JWT_SECRET":        "",
		"SUPABASE_JWKS_URL":          "",
		"SUPABASE_SERVICE_ROLE_KEY":  "test-key",
		"INVITE_HMAC_KEY":            "test-key",
		"BASE_URL":                   "https://example.com",
	})

	errs, warnings := validateConfig()
	if len(errs) != 0 {
		t.Errorf("expected no errors in dev mode, got %d: %v", len(errs), errs)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings when all optional config is set, got %d: %v", len(warnings), warnings)
	}
}

func TestValidateConfig_MissingDatabaseURL(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                "",
		"DATABASE_URL":        "",
		"SUPABASE_URL":        "http://localhost:54321",
		"SUPABASE_JWT_SECRET": "",
		"VAPID_PUBLIC_KEY":    "",
		"VAPID_PRIVATE_KEY":   "",
		"VAPID_SUBJECT":       "",
	})

	errs, _ := validateConfig()

	found := false
	for _, e := range errs {
		if e.envVar == "DATABASE_URL" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for missing DATABASE_URL")
	}
}

func TestValidateConfig_MissingJWTConfig(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                "",
		"DATABASE_URL":        "postgres://localhost/test",
		"SUPABASE_URL":        "",
		"SUPABASE_JWT_SECRET": "",
		"SUPABASE_JWKS_URL":   "",
		"VAPID_PUBLIC_KEY":    "",
		"VAPID_PRIVATE_KEY":   "",
		"VAPID_SUBJECT":       "",
	})

	errs, _ := validateConfig()

	found := false
	for _, e := range errs {
		if e.envVar == "SUPABASE_URL or SUPABASE_JWT_SECRET or SUPABASE_JWKS_URL" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for missing JWT configuration")
	}
}

func TestValidateConfig_SupabaseURLSuffices(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                "",
		"DATABASE_URL":        "postgres://localhost/test",
		"SUPABASE_URL":        "http://localhost:54321",
		"SUPABASE_JWT_SECRET": "",
		"SUPABASE_JWKS_URL":   "",
		"INVITE_HMAC_KEY":     "test-key",
		"BASE_URL":            "https://example.com",
		"VAPID_PUBLIC_KEY":    "BExamplePublicKey",
		"VAPID_PRIVATE_KEY":   "examplePrivateKey",
		"VAPID_SUBJECT":       "mailto:test@example.com",
	})

	errs, _ := validateConfig()
	if len(errs) != 0 {
		t.Errorf("expected no errors when SUPABASE_URL is set, got %d: %v", len(errs), errs)
	}
}

func TestValidateConfig_JWTSecretSuffices(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                "",
		"DATABASE_URL":        "postgres://localhost/test",
		"SUPABASE_URL":        "",
		"SUPABASE_JWT_SECRET": "my-secret",
		"SUPABASE_JWKS_URL":   "",
		"INVITE_HMAC_KEY":     "test-key",
		"BASE_URL":            "https://example.com",
		"VAPID_PUBLIC_KEY":    "BExamplePublicKey",
		"VAPID_PRIVATE_KEY":   "examplePrivateKey",
		"VAPID_SUBJECT":       "mailto:test@example.com",
	})

	errs, _ := validateConfig()
	if len(errs) != 0 {
		t.Errorf("expected no errors when SUPABASE_JWT_SECRET is set, got %d: %v", len(errs), errs)
	}
}

func TestValidateConfig_OptionalWarnings(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                "",
		"DATABASE_URL":        "postgres://localhost/test",
		"SUPABASE_URL":        "http://localhost:54321",
		"INVITE_HMAC_KEY":     "",
		"BASE_URL":            "",
		"VAPID_PUBLIC_KEY":    "",
		"VAPID_PRIVATE_KEY":   "",
		"VAPID_SUBJECT":       "",
	})

	_, warnings := validateConfig()

	wantVars := map[string]bool{
		"INVITE_HMAC_KEY": false,
		"BASE_URL":        false,
	}
	for _, w := range warnings {
		if _, ok := wantVars[w.envVar]; ok {
			wantVars[w.envVar] = true
		}
	}
	for v, found := range wantVars {
		if !found {
			t.Errorf("expected warning for %s", v)
		}
	}
}

func TestValidateConfig_AllValid(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                      "",
		"DATABASE_URL":              "postgres://localhost/test",
		"SUPABASE_URL":              "http://localhost:54321",
		"SUPABASE_SERVICE_ROLE_KEY": "test-service-role-key",
		"INVITE_HMAC_KEY":           "test-key",
		"BASE_URL":                  "https://example.com",
		"VAPID_PUBLIC_KEY":          "BExamplePublicKey",
		"VAPID_PRIVATE_KEY":         "examplePrivateKey",
		"VAPID_SUBJECT":             "mailto:test@example.com",
	})

	errs, warnings := validateConfig()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestValidateConfig_NoVAPIDKeysInProduction_WebPushDisabled(t *testing.T) {
	// When no VAPID vars are set at all, Web Push is simply disabled — no errors.
	setEnvForTest(t, map[string]string{
		"MODE":              "",
		"DATABASE_URL":      "postgres://localhost/test",
		"SUPABASE_URL":      "http://localhost:54321",
		"INVITE_HMAC_KEY":   "test-key",
		"BASE_URL":          "https://example.com",
		"VAPID_PUBLIC_KEY":  "",
		"VAPID_PRIVATE_KEY": "",
		"VAPID_SUBJECT":     "",
	})

	errs, _ := validateConfig()
	for _, e := range errs {
		if e.envVar == "VAPID_PUBLIC_KEY" || e.envVar == "VAPID_PRIVATE_KEY" || e.envVar == "VAPID_SUBJECT" {
			t.Errorf("unexpected VAPID error when no VAPID vars are set (Web Push should be disabled): %s: %s", e.envVar, e.message)
		}
	}
}

func TestValidateConfig_PartialVAPIDInProduction_SubjectOnly(t *testing.T) {
	// If only VAPID_SUBJECT is set, the keys are missing — error.
	setEnvForTest(t, map[string]string{
		"MODE":              "",
		"DATABASE_URL":      "postgres://localhost/test",
		"SUPABASE_URL":      "http://localhost:54321",
		"VAPID_PUBLIC_KEY":  "",
		"VAPID_PRIVATE_KEY": "",
		"VAPID_SUBJECT":     "mailto:test@example.com",
	})

	errs, _ := validateConfig()

	foundPub := false
	foundPriv := false
	for _, e := range errs {
		if e.envVar == "VAPID_PUBLIC_KEY" {
			foundPub = true
		}
		if e.envVar == "VAPID_PRIVATE_KEY" {
			foundPriv = true
		}
	}
	if !foundPub {
		t.Error("expected error for missing VAPID_PUBLIC_KEY when VAPID_SUBJECT is set")
	}
	if !foundPriv {
		t.Error("expected error for missing VAPID_PRIVATE_KEY when VAPID_SUBJECT is set")
	}
}

func TestValidateConfig_VAPIDKeysNotRequiredInDevMode(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":              "development",
		"VAPID_PUBLIC_KEY":  "",
		"VAPID_PRIVATE_KEY": "",
		"VAPID_SUBJECT":     "",
	})

	errs, _ := validateConfig()
	vapidVars := map[string]bool{
		"VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY": true,
		"VAPID_PUBLIC_KEY":                       true,
		"VAPID_PRIVATE_KEY":                      true,
		"VAPID_SUBJECT":                          true,
	}
	for _, e := range errs {
		if vapidVars[e.envVar] {
			t.Errorf("unexpected VAPID error in dev mode: %s: %s", e.envVar, e.message)
		}
	}
}

func TestValidateConfig_PartialVAPIDKeysInProduction_MissingPrivate(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":              "",
		"DATABASE_URL":      "postgres://localhost/test",
		"SUPABASE_URL":      "http://localhost:54321",
		"VAPID_PUBLIC_KEY":  "BExamplePublicKey",
		"VAPID_PRIVATE_KEY": "", // missing private key
		"VAPID_SUBJECT":     "mailto:test@example.com",
	})

	errs, _ := validateConfig()

	found := false
	for _, e := range errs {
		if e.envVar == "VAPID_PRIVATE_KEY" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error specifically for missing VAPID_PRIVATE_KEY")
	}
}

func TestValidateConfig_PartialVAPIDKeysInProduction_MissingPublic(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":              "",
		"DATABASE_URL":      "postgres://localhost/test",
		"SUPABASE_URL":      "http://localhost:54321",
		"VAPID_PUBLIC_KEY":  "",
		"VAPID_PRIVATE_KEY": "examplePrivateKey", // missing public key
		"VAPID_SUBJECT":     "mailto:test@example.com",
	})

	errs, _ := validateConfig()

	found := false
	for _, e := range errs {
		if e.envVar == "VAPID_PUBLIC_KEY" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error specifically for missing VAPID_PUBLIC_KEY")
	}
}

func TestValidateConfig_VAPIDSubjectMustBeContactURI(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":              "",
		"DATABASE_URL":      "postgres://localhost/test",
		"SUPABASE_URL":      "http://localhost:54321",
		"VAPID_PUBLIC_KEY":  "BExamplePublicKey",
		"VAPID_PRIVATE_KEY": "examplePrivateKey",
		"VAPID_SUBJECT":     "admin@example.com", // missing mailto: or https:// prefix
	})

	errs, _ := validateConfig()

	found := false
	for _, e := range errs {
		if e.envVar == "VAPID_SUBJECT" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for VAPID_SUBJECT without mailto: or https:// prefix")
	}
}

func TestValidateConfig_BillingEnabled_RequiresStripeKeysInProd(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                "",
		"DATABASE_URL":        "postgres://localhost/test",
		"SUPABASE_URL":        "http://localhost:54321",
		"BILLING_ENABLED":     "true",
		"STRIPE_SECRET_KEY":   "",
		"STRIPE_WEBHOOK_SECRET": "",
		"BASE_URL":            "",
		"VAPID_PUBLIC_KEY":    "",
		"VAPID_PRIVATE_KEY":   "",
		"VAPID_SUBJECT":       "",
	})

	errs, _ := validateConfig()

	wantErrors := map[string]bool{
		"STRIPE_SECRET_KEY":    false,
		"STRIPE_WEBHOOK_SECRET": false,
		"BASE_URL":             false,
	}
	for _, e := range errs {
		if _, ok := wantErrors[e.envVar]; ok {
			wantErrors[e.envVar] = true
		}
	}
	for v, found := range wantErrors {
		if !found {
			t.Errorf("expected error for missing %s when BILLING_ENABLED=true", v)
		}
	}
}

func TestValidateConfig_BillingEnabled_NoErrorsWhenConfigured(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                      "",
		"DATABASE_URL":              "postgres://localhost/test",
		"SUPABASE_URL":              "http://localhost:54321",
		"SUPABASE_SERVICE_ROLE_KEY": "test-key",
		"BILLING_ENABLED":           "true",
		"STRIPE_SECRET_KEY":         "sk_test_xxx",
		"STRIPE_WEBHOOK_SECRET":     "whsec_xxx",
		"STRIPE_PRICE_ID_REQUEST":   "price_xxx",
		"BASE_URL":                  "https://example.com",
		"INVITE_HMAC_KEY":           "test-key",
		"VAPID_PUBLIC_KEY":          "",
		"VAPID_PRIVATE_KEY":         "",
		"VAPID_SUBJECT":             "",
	})

	errs, warnings := validateConfig()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestValidateConfig_VAPIDSubjectAcceptsHTTPS(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":              "",
		"DATABASE_URL":      "postgres://localhost/test",
		"SUPABASE_URL":      "http://localhost:54321",
		"VAPID_PUBLIC_KEY":  "BExamplePublicKey",
		"VAPID_PRIVATE_KEY": "examplePrivateKey",
		"VAPID_SUBJECT":     "https://example.com/contact",
		"INVITE_HMAC_KEY":   "test-key",
		"BASE_URL":          "https://example.com",
	})

	errs, _ := validateConfig()
	for _, e := range errs {
		if e.envVar == "VAPID_SUBJECT" {
			t.Errorf("unexpected VAPID_SUBJECT error with https:// prefix: %s", e.message)
		}
	}
}
