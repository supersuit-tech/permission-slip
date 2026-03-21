package api

import (
	"log/slog"
	"net/http"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/notify"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	pstripe "github.com/supersuit-tech/permission-slip-web/stripe"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

// Deps holds shared dependencies for API route handlers.
type Deps struct {
	DB                     db.DBTX                 // nil when running without a database
	Vault                  vault.VaultStore        // credential secret encryption; nil returns 503 on credential endpoints
	SupabaseJWTSecret      string                  // HMAC-SHA256 secret for HS256 JWTs (Supabase CLI v1 / test env)
	SupabaseJWKSURL        string                  // JWKS endpoint for ES256 JWTs (Supabase CLI v2+), e.g. "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	JWKSCache              *JWKSCache              // JWKS key cache; initialized once at startup when SupabaseJWKSURL is set
	SupabaseURL            string                  // Supabase project URL (e.g. "http://127.0.0.1:54321"); used for Admin API calls
	SupabaseServiceRoleKey string                  // Supabase service_role key; required for Admin API calls (e.g. deleting auth users)
	BaseURL                string                  // Public base URL (e.g. "https://app.permissionslip.dev"); used to construct invite URLs
	InviteHMACKey          string                  // HMAC key for hashing short codes (invite codes, confirmation codes); if empty, falls back to plain SHA-256
	Notifier               *notify.Dispatcher      // notification fan-out; nil means notifications are disabled
	VAPIDPublicKey         string                  // VAPID public key for Web Push; empty if not configured
	Connectors             *connectors.Registry    // connector execution registry; nil means no connectors are available
	OAuthProviders         *oauth.Registry         // OAuth provider registry; nil means OAuth is not available
	OAuthRedirectBaseURL   string                  // Public base URL for OAuth callbacks (e.g. "https://app.permissionslip.dev"); falls back to BaseURL
	OAuthStateSecret       string                  // HMAC-SHA256 secret for signing OAuth CSRF state tokens; if empty, falls back to SupabaseJWTSecret
	Stripe                 *pstripe.Client         // Stripe API client; nil when billing is disabled or Stripe keys not set
	CouponSecret           string                  // HMAC key for free-pro coupons; empty disables POST /billing/redeem-coupon
	BillingEnabled         bool                    // true when BILLING_ENABLED=true; gates Stripe, metering, and billing endpoints
	SMSEnabled             bool                    // true when SMS sender is configured AND SMS_NOTIFICATIONS_HIDDEN != "true"; gates SMS preference visibility
	DevMode                bool                    // true when MODE=development; disables rate limiting
	RateLimiter            *RateLimiter            // pre-auth rate limiter (per-IP + global); nil disables
	AgentRateLimiter       *RateLimiter            // post-auth rate limiter (per verified agent); nil disables
	TrustedProxyHeader     string                  // header to read client IP from behind a reverse proxy (e.g. "Fly-Client-IP"); empty uses RemoteAddr
	AllowedOrigins         []string                // allowed CORS origins; empty means cross-origin requests are blocked
	Logger                 *slog.Logger            // structured logger for request logging; if nil, request logging is skipped
	ApprovalEvents         *ApprovalEventBroker    // SSE broker for real-time approval notifications; nil disables SSE
	SlackSigningSecret     string                  // Slack signing secret for verifying Events API webhook signatures; empty disables Slack events
	EventBroker            *connectors.EventBroker // connector event dispatch; nil means inbound events are accepted but not processed
}

// NewRouter returns an http.Handler that serves all /api/v1/ routes.
// The returned handler includes the TraceIDMiddleware, RequestLoggerMiddleware,
// and RateLimitMiddleware.
//
// Rate limiting is scoped to /api/v1/ intentionally. Endpoints outside this
// router (/api/health for load-balancer probes, /invite/ for user-facing
// onboarding pages) are excluded to avoid interfering with health checks
// or blocking low-volume user-facing flows.
//
// Each domain registers its own routes via a Register* function defined
// in its own file (e.g., agents.go, approvals.go). This keeps the router
// small and lets multiple phases add endpoints without merge conflicts.
func NewRouter(deps *Deps) http.Handler {
	mux := http.NewServeMux()

	// Each domain file self-registers via init() + RegisterRouteGroup().
	// See router_registry.go for the registry.
	for _, register := range routeGroups {
		register(mux, deps)
	}
	// NOTE: Billing webhook routes are registered on the top-level mux in
	// main.go, NOT here. They must bypass auth and rate-limiting middleware
	// because Stripe verifies requests via signature, not Bearer tokens.

	var handler http.Handler = mux
	handler = RateLimitMiddleware(deps.RateLimiter, deps.DevMode, deps.TrustedProxyHeader)(handler)
	if deps.Logger != nil {
		handler = RequestLoggerMiddleware(deps.Logger, deps.TrustedProxyHeader)(handler)
	}
	// Middleware execution order (outermost → innermost):
	//   TraceIDMiddleware → sentryhttp.Handler → SentryTraceIDMiddleware → ...
	//
	// TraceID generates the trace ID first, sentryhttp puts a Sentry hub in
	// context, then SentryTraceIDMiddleware tags the hub with the trace ID.
	// sentryhttp also captures panics with full stack traces. Repanic: true
	// re-panics so net/http or an upstream supervisor can also observe them.
	sentryHandler := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	})
	handler = SentryTraceIDMiddleware(handler)
	handler = sentryHandler.Handle(handler)
	handler = TraceIDMiddleware(handler)
	return handler
}
