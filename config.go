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
	// In dev, just set STRIPE_SECRET_KEY etc. to your test-mode keys (sk_test_...).
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
				errs = append(errs, configError{
					envVar:  "STRIPE_WEBHOOK_SECRET",
					message: "required when BILLING_ENABLED=true (webhook signature verification prevents spoofed events)",
				})
			}
			if !hasPriceID {
				warnings = append(warnings, configError{
					envVar:  "STRIPE_PRICE_ID_REQUEST",
					message: "not set; checkout session creation will fail without a metered price ID",
				})
			}
			if os.Getenv("BASE_URL") == "" {
				errs = append(errs, configError{
					envVar:  "BASE_URL",
					message: "required when BILLING_ENABLED=true (checkout session success/cancel redirect URLs need a base URL)",
				})
			}
		}
	}

	// OAuth — warn when built-in provider client credentials are not set.
	// These are non-fatal because BYOA still works, but the built-in
	// connectors won't be usable without them.
	hasGoogleClientID := os.Getenv("GOOGLE_CLIENT_ID") != ""
	hasGoogleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET") != ""
	hasMicrosoftClientID := os.Getenv("MICROSOFT_CLIENT_ID") != ""
	hasMicrosoftClientSecret := os.Getenv("MICROSOFT_CLIENT_SECRET") != ""
	hasSalesforceClientID := os.Getenv("SALESFORCE_CLIENT_ID") != ""
	hasSalesforceClientSecret := os.Getenv("SALESFORCE_CLIENT_SECRET") != ""
	if !hasGoogleClientID || !hasGoogleClientSecret {
		warnings = append(warnings, configError{
			envVar:  "GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET",
			message: "not set; Google OAuth connector will require BYOA credentials (see docs/oauth-setup.md)",
		})
	}
	if !hasMicrosoftClientID || !hasMicrosoftClientSecret {
		warnings = append(warnings, configError{
			envVar:  "MICROSOFT_CLIENT_ID / MICROSOFT_CLIENT_SECRET",
			message: "not set; Microsoft OAuth connector will require BYOA credentials (see docs/oauth-setup.md)",
		})
	}
	if !hasSalesforceClientID || !hasSalesforceClientSecret {
		warnings = append(warnings, configError{
			envVar:  "SALESFORCE_CLIENT_ID / SALESFORCE_CLIENT_SECRET",
			message: "not set; Salesforce OAuth connector will require BYOA credentials (see docs/oauth-setup.md)",
		})
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
		msg := "not set; invite URLs will not be generated"
		if os.Getenv("OAUTH_REDIRECT_BASE_URL") == "" {
			msg += " and OAuth callback URLs cannot be constructed (set BASE_URL or OAUTH_REDIRECT_BASE_URL)"
		}
		warnings = append(warnings, configError{
			envVar:  "BASE_URL",
			message: msg,
		})
	}

	// OAuth state secret — required in production when SUPABASE_JWT_SECRET is
	// not set (e.g. JWKS-based auth with Supabase CLI v2+). Without either
	// secret, OAuth CSRF state tokens cannot be signed and all flows will fail.
	if os.Getenv("OAUTH_STATE_SECRET") == "" && !hasJWTSecret {
		ce := configError{
			envVar:  "OAUTH_STATE_SECRET",
			message: "not set and SUPABASE_JWT_SECRET is empty — OAuth flows will fail (generate with: openssl rand -hex 32)",
		}
		if devMode {
			warnings = append(warnings, ce)
		} else {
			errs = append(errs, ce)
		}
	}

	// Notification email — warn if provider is set but required companion vars are missing.
	emailProvider := os.Getenv("NOTIFICATION_EMAIL_PROVIDER")
	if emailProvider == "twilio-sendgrid" {
		if os.Getenv("SENDGRID_API_KEY") == "" {
			warnings = append(warnings, configError{
				envVar:  "SENDGRID_API_KEY",
				message: "not set; NOTIFICATION_EMAIL_PROVIDER=twilio-sendgrid but API key is missing — email will be disabled",
			})
		}
		if os.Getenv("NOTIFICATION_EMAIL_FROM") == "" {
			warnings = append(warnings, configError{
				envVar:  "NOTIFICATION_EMAIL_FROM",
				message: "not set; required for sending email notifications",
			})
		}
	} else if emailProvider == "smtp" {
		if os.Getenv("SMTP_HOST") == "" {
			warnings = append(warnings, configError{
				envVar:  "SMTP_HOST",
				message: "not set; NOTIFICATION_EMAIL_PROVIDER=smtp but SMTP host is missing — email will be disabled",
			})
		}
		if os.Getenv("NOTIFICATION_EMAIL_FROM") == "" {
			warnings = append(warnings, configError{
				envVar:  "NOTIFICATION_EMAIL_FROM",
				message: "not set; required for sending email notifications",
			})
		}
	}

	// SMS (Amazon SNS) — AWS_REGION is required; credentials are optional
	// (the AWS SDK falls back to IAM roles, shared config, etc.).
	hasAWSRegion := os.Getenv("AWS_REGION") != ""
	hasAWSKeyID := os.Getenv("AWS_ACCESS_KEY_ID") != ""
	hasAWSSecret := os.Getenv("AWS_SECRET_ACCESS_KEY") != ""
	if !hasAWSRegion && (hasAWSKeyID || hasAWSSecret) {
		warnings = append(warnings, configError{
			envVar:  "AWS_REGION",
			message: "not set; AWS credentials are configured but AWS_REGION is required for SMS (SNS)",
		})
	}
	if hasAWSRegion && (hasAWSKeyID != hasAWSSecret) {
		missing := "AWS_SECRET_ACCESS_KEY"
		if !hasAWSKeyID {
			missing = "AWS_ACCESS_KEY_ID"
		}
		warnings = append(warnings, configError{
			envVar:  missing,
			message: "not set; AWS credentials are partially configured — set both AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY, or neither (to use IAM roles)",
		})
	}

	return errs, warnings
}
