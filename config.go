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
	billingEnabled := os.Getenv("BILLING_ENABLED") == "true"
	if billingEnabled {
		hasStripeKey := os.Getenv("STRIPE_SECRET_KEY") != ""
		hasWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET") != ""
		hasPriceID := os.Getenv("STRIPE_PRICE_ID_REQUEST") != ""

		if !devMode {
			if !hasStripeKey {
				errs = append(errs, configError{
					envVar:  "STRIPE_SECRET_KEY",
					message: "required when BILLING_ENABLED=true (Stripe API key for billing)",
				})
			}
			if !hasWebhookSecret {
				warnings = append(warnings, configError{
					envVar:  "STRIPE_WEBHOOK_SECRET",
					message: "not set; Stripe webhook signature verification will be disabled",
				})
			}
			if !hasPriceID {
				warnings = append(warnings, configError{
					envVar:  "STRIPE_PRICE_ID_REQUEST",
					message: "not set; checkout session creation will fail without a metered price ID",
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
