package api

import (
	"crypto/ecdsa"
	"log/slog"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/notify"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

// Deps holds shared dependencies for API route handlers.
type Deps struct {
	DB                db.DBTX          // nil when running without a database
	Vault             vault.VaultStore // credential secret encryption; nil returns 503 on credential endpoints
	SupabaseJWTSecret string           // HMAC-SHA256 secret for HS256 JWTs (Supabase CLI v1 / test env)
	SupabaseJWKSURL   string           // JWKS endpoint for ES256 JWTs (Supabase CLI v2+), e.g. "http://127.0.0.1:54321/auth/v1/.well-known/jwks.json"
	JWKSCache         *JWKSCache       // JWKS key cache; initialized once at startup when SupabaseJWKSURL is set
	BaseURL           string           // Public base URL (e.g. "https://app.permissionslip.dev"); used to construct invite URLs
	InviteHMACKey     string           // HMAC key for hashing short codes (invite codes, confirmation codes); if empty, falls back to plain SHA-256
	Notifier              *notify.Dispatcher   // notification fan-out; nil means notifications are disabled
	VAPIDPublicKey        string               // VAPID public key for Web Push; empty if not configured
	Connectors            *connectors.Registry // connector execution registry; nil means no connectors are available
	DevMode               bool                 // true when MODE=development; disables rate limiting
	RateLimiter           *RateLimiter         // pre-auth rate limiter (per-IP + global); nil disables
	AgentRateLimiter      *RateLimiter         // post-auth rate limiter (per verified agent); nil disables
	TrustedProxyHeader    string               // header to read client IP from behind a reverse proxy (e.g. "Fly-Client-IP"); empty uses RemoteAddr
	AllowedOrigins        []string             // allowed CORS origins; empty means cross-origin requests are blocked
	ActionTokenSigningKey *ecdsa.PrivateKey     // ECDSA P-256 private key for signing action tokens (ES256); auto-generated at startup
	ActionTokenKeyID      string               // Key ID for the action token signing key; used in JWT "kid" header
	Logger                *slog.Logger         // structured logger for request logging; if nil, request logging is skipped
	ApprovalEvents        *ApprovalEventBroker // SSE broker for real-time approval notifications; nil disables SSE
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

	RegisterActionConfigRoutes(mux, deps)
	RegisterActionConfigTemplateRoutes(mux, deps)
	RegisterActionExecuteRoutes(mux, deps)
	RegisterAgentApprovalRoutes(mux, deps)
	RegisterAgentRoutes(mux, deps)
	RegisterAgentConnectorRoutes(mux, deps)
	RegisterAgentStandingApprovalRoutes(mux, deps)
	RegisterApprovalRoutes(mux, deps)
	RegisterCapabilityRoutes(mux, deps)
	RegisterAuditEventRoutes(mux, deps)
	RegisterConnectorRoutes(mux, deps)
	RegisterCredentialRoutes(mux, deps)
	RegisterOnboardingRoutes(mux, deps)
	RegisterProfileRoutes(mux, deps)
	RegisterRegistrationInviteRoutes(mux, deps)
	RegisterRegistrationRoutes(mux, deps)
	RegisterPushSubscriptionRoutes(mux, deps)
	RegisterStandingApprovalRoutes(mux, deps)
	RegisterApprovalEventRoutes(mux, deps)

	var handler http.Handler = mux
	handler = RateLimitMiddleware(deps.RateLimiter, deps.DevMode, deps.TrustedProxyHeader)(handler)
	if deps.Logger != nil {
		handler = RequestLoggerMiddleware(deps.Logger, deps.TrustedProxyHeader)(handler)
	}
	handler = TraceIDMiddleware(handler)
	return handler
}
