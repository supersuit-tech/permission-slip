package main

import (
	"os"
	"strings"
)

// configError represents a missing or invalid configuration value.
type configError struct {
	envVar  string
	message string
}

// validateConfig checks that all required environment variables are set.
// It returns a list of errors for missing required config and a list of
// warnings for optional but recommended config.
//
// Warnings are emitted in all modes. Errors are only emitted in production
// (non-development) mode — in development, missing required config is tolerated.
func validateConfig() (errs []configError, warnings []configError) {
	devMode := os.Getenv("MODE") == "development"

	// Required for API mode — the server can't serve API requests without a database.
	if !devMode && os.Getenv("DATABASE_URL") == "" {
		errs = append(errs, configError{
			envVar:  "DATABASE_URL",
			message: "required for database connectivity",
		})
	}

	// Required — JWT authentication won't work without at least one signing method.
	hasJWTSecret := os.Getenv("SUPABASE_JWT_SECRET") != ""
	hasJWKSURL := os.Getenv("SUPABASE_JWKS_URL") != ""
	hasSupabaseURL := os.Getenv("SUPABASE_URL") != ""
	if !devMode && !hasJWTSecret && !hasJWKSURL && !hasSupabaseURL {
		errs = append(errs, configError{
			envVar:  "SUPABASE_URL or SUPABASE_JWT_SECRET or SUPABASE_JWKS_URL",
			message: "required for JWT authentication (set SUPABASE_URL for ES256, SUPABASE_JWKS_URL for JWKS-based verification, or SUPABASE_JWT_SECRET for HS256)",
		})
	}

	// Web Push (VAPID) — optional channel, but if partially configured, error.
	// If no VAPID vars are set, Web Push is simply disabled. If some are set,
	// all must be set to avoid misconfiguration.
	// Trim whitespace to handle common copy-paste issues (trailing newlines, etc.).
	hasVAPIDPublicKey := strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY")) != ""
	hasVAPIDPrivateKey := strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY")) != ""
	vapidSubject := strings.TrimSpace(os.Getenv("VAPID_SUBJECT"))
	hasVAPIDSubject := vapidSubject != ""
	anyVAPID := hasVAPIDPublicKey || hasVAPIDPrivateKey || hasVAPIDSubject
	if !devMode && anyVAPID {
		if !hasVAPIDPublicKey {
			errs = append(errs, configError{
				envVar:  "VAPID_PUBLIC_KEY",
				message: "missing (other VAPID vars are set; all three are required for Web Push); generate keys with: make generate-vapid-keys",
			})
		}
		if !hasVAPIDPrivateKey {
			errs = append(errs, configError{
				envVar:  "VAPID_PRIVATE_KEY",
				message: "missing (other VAPID vars are set; all three are required for Web Push); generate keys with: make generate-vapid-keys",
			})
		}
		if !hasVAPIDSubject {
			errs = append(errs, configError{
				envVar:  "VAPID_SUBJECT",
				message: "missing (other VAPID vars are set; required by the Web Push spec, e.g. mailto:admin@mycompany.com)",
			})
		} else if !strings.HasPrefix(vapidSubject, "mailto:") && !strings.HasPrefix(vapidSubject, "https://") {
			errs = append(errs, configError{
				envVar:  "VAPID_SUBJECT",
				message: "must start with \"mailto:\" or \"https://\" per the Web Push spec (e.g. mailto:admin@mycompany.com)",
			})
		}
	}

	// Stripe — required when BILLING_ENABLED=true and in production.
	// If billing is enabled but keys are missing, warn (dev) or error (prod).
	//
	// When STRIPE_TEST_MODE=true (dev only), the server reads _TEST-suffixed
	// env vars instead. Validate whichever set will actually be used.
	billingEnabled := os.Getenv("BILLING_ENABLED") == "true"
	if billingEnabled {
		stripeTestMode := os.Getenv("STRIPE_TEST_MODE") == "true"

		// Determine which env vars to validate based on test mode.
		keyVar := "STRIPE_SECRET_KEY"
		webhookVar := "STRIPE_WEBHOOK_SECRET"
		priceVar := "STRIPE_PRICE_ID_REQUEST"
		if stripeTestMode {
			keyVar = "STRIPE_SECRET_KEY_TEST"
			webhookVar = "STRIPE_WEBHOOK_SECRET_TEST"
			priceVar = "STRIPE_PRICE_ID_REQUEST_TEST"
		}
		hasStripeKey := os.Getenv(keyVar) != ""
		hasWebhookSecret := os.Getenv(webhookVar) != ""
		hasPriceID := os.Getenv(priceVar) != ""

		// STRIPE_TEST_MODE should only be used in development.
		if stripeTestMode && !devMode {
			errs = append(errs, configError{
				envVar:  "STRIPE_TEST_MODE",
				message: "must not be set to true in production — use live Stripe keys instead",
			})
		}

		if !devMode {
			if !hasStripeKey {
				errs = append(errs, configError{
					envVar:  keyVar,
					message: "required when BILLING_ENABLED=true (Stripe API key for billing)",
				})
			}
			if !hasWebhookSecret {
				errs = append(errs, configError{
					envVar:  webhookVar,
					message: "required when BILLING_ENABLED=true (webhook signature verification prevents spoofed events)",
				})
			}
			if !hasPriceID {
				warnings = append(warnings, configError{
					envVar:  priceVar,
					message: "not set; checkout session creation will fail without a metered price ID",
				})
			}
			if os.Getenv("BASE_URL") == "" {
				errs = append(errs, configError{
					envVar:  "BASE_URL",
					message: "required when BILLING_ENABLED=true (checkout session success/cancel redirect URLs need a base URL)",
				})
			}
		} else if stripeTestMode {
			// In dev mode with test mode, warn about missing test keys.
			if !hasStripeKey {
				warnings = append(warnings, configError{
					envVar:  keyVar,
					message: "not set; STRIPE_TEST_MODE is true but test secret key is missing",
				})
			}
			if !hasWebhookSecret {
				warnings = append(warnings, configError{
					envVar:  webhookVar,
					message: "not set; STRIPE_TEST_MODE is true but test webhook secret is missing",
				})
			}
		}
	}

	// Optional but recommended — warn in all modes.
	if os.Getenv("SUPABASE_SERVICE_ROLE_KEY") == "" {
		warnings = append(warnings, configError{
			envVar:  "SUPABASE_SERVICE_ROLE_KEY",
			message: "not set; account deletion will not remove the Supabase auth user (recommended: set for production)",
		})
	}
	if os.Getenv("INVITE_HMAC_KEY") == "" {
		warnings = append(warnings, configError{
			envVar:  "INVITE_HMAC_KEY",
			message: "not set; invite codes will use plain SHA-256 (recommended: set for production)",
		})
	}
	if os.Getenv("BASE_URL") == "" {
		warnings = append(warnings, configError{
			envVar:  "BASE_URL",
			message: "not set; invite URLs will not be generated",
		})
	}

	return errs, warnings
}
