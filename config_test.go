package main

import (
	"os"
	"testing"
)

// testSecretEncryptionKeyB64 is 32 zero-bytes, base64-encoded (tests only).
const testSecretEncryptionKeyB64 = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

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
		"MODE":               "development",
		"DATABASE_PATH":      "",
		"JWT_SIGNING_SECRET": "",
		"INVITE_HMAC_KEY":    "",
		"BASE_URL":           "",
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
		"MODE":                     "development",
		"DATABASE_PATH":            "",
		"JWT_SIGNING_SECRET":       "test-jwt-signing-secret-32chars-min!",
		"INVITE_HMAC_KEY":          "test-invite-hmac-key-at-least-32c!",
		"BASE_URL":                 "https://example.com",
		"OAUTH_STATE_SECRET":       "test-oauth-state-secret-32chars!",
		"GOOGLE_CLIENT_ID":         "test-google-id",
		"GOOGLE_CLIENT_SECRET":     "test-google-secret",
		"MICROSOFT_CLIENT_ID":      "test-msft-id",
		"MICROSOFT_CLIENT_SECRET":  "test-msft-secret",
		"SALESFORCE_CLIENT_ID":     "test-sf-id",
		"SALESFORCE_CLIENT_SECRET": "test-sf-secret",
		"AWS_REGION":               "",
		"AWS_ACCESS_KEY_ID":        "",
		"AWS_SECRET_ACCESS_KEY":    "",
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
		"MODE":               "",
		"DATABASE_PATH":      "",
		"JWT_SIGNING_SECRET": "test-jwt-signing-secret-32chars-min!",
	})

	errs, _ := validateConfig()

	found := false
	for _, e := range errs {
		if e.envVar == "DATABASE_PATH" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for missing DATABASE_PATH")
	}
}

func TestValidateConfig_MissingJWTSigningSecret(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                  "",
		"DATABASE_PATH":         "/tmp/test.db",
		"SECRET_ENCRYPTION_KEY": testSecretEncryptionKeyB64,
		"JWT_SIGNING_SECRET":    "",
	})

	errs, _ := validateConfig()

	found := false
	for _, e := range errs {
		if e.envVar == "JWT_SIGNING_SECRET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for missing JWT_SIGNING_SECRET")
	}
}

func TestValidateConfig_MissingSecretEncryptionKeyInProduction(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                  "",
		"DATABASE_PATH":         "/tmp/test.db",
		"SECRET_ENCRYPTION_KEY": "",
		"JWT_SIGNING_SECRET":    "test-jwt-signing-secret-32chars-min!",
		"INVITE_HMAC_KEY":       "test-invite-hmac-key-at-least-32c!",
		"BASE_URL":              "https://example.com",
	})

	errs, _ := validateConfig()
	found := false
	for _, e := range errs {
		if e.envVar == "SECRET_ENCRYPTION_KEY" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for missing SECRET_ENCRYPTION_KEY when DATABASE_PATH is set")
	}
}

func TestValidateConfig_InvalidSecretEncryptionKeyInProduction(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                  "",
		"DATABASE_PATH":         "/tmp/test.db",
		"SECRET_ENCRYPTION_KEY": "not-valid-base64!!!",
		"JWT_SIGNING_SECRET":    "test-jwt-signing-secret-32chars-min!",
		"INVITE_HMAC_KEY":       "test-invite-hmac-key-at-least-32c!",
		"BASE_URL":              "https://example.com",
	})

	errs, _ := validateConfig()
	found := false
	for _, e := range errs {
		if e.envVar == "SECRET_ENCRYPTION_KEY" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error for invalid SECRET_ENCRYPTION_KEY")
	}
}

func TestValidateConfig_JWTSigningSecretSuffices(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                  "",
		"DATABASE_PATH":         "/tmp/test.db",
		"SECRET_ENCRYPTION_KEY": testSecretEncryptionKeyB64,
		"JWT_SIGNING_SECRET":    "my-secret-that-is-32-chars-long!",
		"INVITE_HMAC_KEY":       "test-invite-hmac-key-at-least-32c!",
		"BASE_URL":              "https://example.com",
	})

	errs, _ := validateConfig()
	if len(errs) != 0 {
		t.Errorf("expected no errors when JWT_SIGNING_SECRET is set, got %d: %v", len(errs), errs)
	}
}

func TestValidateConfig_OptionalWarnings(t *testing.T) {
	// INVITE_HMAC_KEY is required in production (errors out), so this test
	// runs in development mode where it falls back to a warning. BASE_URL
	// is always optional.
	setEnvForTest(t, map[string]string{
		"MODE":                  "development",
		"DATABASE_PATH":         "/tmp/test.db",
		"SECRET_ENCRYPTION_KEY": testSecretEncryptionKeyB64,
		"JWT_SIGNING_SECRET":    "test-jwt-signing-secret-32chars-min!",
		"INVITE_HMAC_KEY":       "",
		"BASE_URL":              "",
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
		"MODE":                     "",
		"DATABASE_PATH":            "/tmp/test.db",
		"SECRET_ENCRYPTION_KEY":    testSecretEncryptionKeyB64,
		"JWT_SIGNING_SECRET":       "test-jwt-signing-secret-32chars-min!",
		"INVITE_HMAC_KEY":          "test-invite-hmac-key-at-least-32c!",
		"BASE_URL":                 "https://example.com",
		"OAUTH_STATE_SECRET":       "test-oauth-state-secret-32chars!",
		"GOOGLE_CLIENT_ID":         "test-google-id",
		"GOOGLE_CLIENT_SECRET":     "test-google-secret",
		"MICROSOFT_CLIENT_ID":      "test-msft-id",
		"MICROSOFT_CLIENT_SECRET":  "test-msft-secret",
		"SALESFORCE_CLIENT_ID":     "test-sf-id",
		"SALESFORCE_CLIENT_SECRET": "test-sf-secret",
		"AWS_REGION":               "",
		"AWS_ACCESS_KEY_ID":        "",
		"AWS_SECRET_ACCESS_KEY":    "",
	})

	errs, warnings := validateConfig()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d: %v", len(warnings), warnings)
	}
}

func TestValidateConfig_OAuthWarnings_MissingGoogleCredentials(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                     "development",
		"GOOGLE_CLIENT_ID":         "",
		"GOOGLE_CLIENT_SECRET":     "",
		"MICROSOFT_CLIENT_ID":      "test-msft-id",
		"MICROSOFT_CLIENT_SECRET":  "test-msft-secret",
		"SALESFORCE_CLIENT_ID":     "test-sf-id",
		"SALESFORCE_CLIENT_SECRET": "test-sf-secret",
		"JWT_SIGNING_SECRET":       "test-jwt-signing-secret-32chars-min!",
		"INVITE_HMAC_KEY":          "test-invite-hmac-key-at-least-32c!",
		"BASE_URL":                 "https://example.com",
	})

	_, warnings := validateConfig()

	found := false
	for _, w := range warnings {
		if w.envVar == "GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning for missing Google OAuth credentials")
	}

	// Microsoft should NOT have a warning.
	for _, w := range warnings {
		if w.envVar == "MICROSOFT_CLIENT_ID / MICROSOFT_CLIENT_SECRET" {
			t.Error("unexpected warning for Microsoft OAuth when credentials are set")
		}
	}
}

func TestValidateConfig_OAuthWarnings_MissingMicrosoftCredentials(t *testing.T) {
	setEnvForTest(t, map[string]string{
		"MODE":                     "development",
		"GOOGLE_CLIENT_ID":         "test-google-id",
		"GOOGLE_CLIENT_SECRET":     "test-google-secret",
		"MICROSOFT_CLIENT_ID":      "",
		"MICROSOFT_CLIENT_SECRET":  "",
		"SALESFORCE_CLIENT_ID":     "test-sf-id",
		"SALESFORCE_CLIENT_SECRET": "test-sf-secret",
		"JWT_SIGNING_SECRET":       "test-jwt-signing-secret-32chars-min!",
		"INVITE_HMAC_KEY":          "test-invite-hmac-key-at-least-32c!",
		"BASE_URL":                 "https://example.com",
	})

	_, warnings := validateConfig()

	found := false
	for _, w := range warnings {
		if w.envVar == "MICROSOFT_CLIENT_ID / MICROSOFT_CLIENT_SECRET" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning for missing Microsoft OAuth credentials")
	}

	// Google should NOT have a warning.
	for _, w := range warnings {
		if w.envVar == "GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET" {
			t.Error("unexpected warning for Google OAuth when credentials are set")
		}
	}
}
