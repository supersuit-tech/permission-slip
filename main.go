package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"embed"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/joho/godotenv"
	"github.com/supersuit-tech/permission-slip-web/api"
	"github.com/supersuit-tech/permission-slip-web/connectors"
	ghconnector "github.com/supersuit-tech/permission-slip-web/connectors/github"
	"github.com/supersuit-tech/permission-slip-web/connectors/slack"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/notify"
	"github.com/supersuit-tech/permission-slip-web/notify/webpush"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

//go:embed all:frontend/dist
var distFS embed.FS

// version is set at build time via -ldflags.
// Example: go build -ldflags "-X main.version=abc1234" -o bin/server .
var version = "dev"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Set up structured JSON logger for production use.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize Sentry error tracking. When SENTRY_DSN is not set the call
	// is a no-op — sentry.Init returns nil error and the SDK remains inactive,
	// so no events are sent. This keeps dev/test environments clean.
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		environment := os.Getenv("MODE")
		if environment == "" {
			environment = "production"
		}
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      environment,
			Release:          version,
			AttachStacktrace: true,
			// Sample 100% of error events. Adjust if volume becomes a concern.
			SampleRate: 1.0,
			// Scrub sensitive data before sending events to Sentry.
			// sentryhttp captures request headers AND a separate Cookies field;
			// both must be cleared to prevent PII/credential leakage.
			BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				if event.Request != nil {
					for _, h := range []string{"Authorization", "Cookie", "X-Api-Key"} {
						delete(event.Request.Headers, h)
					}
					event.Request.Cookies = ""
				}
				return event
			},
		}); err != nil {
			logger.Error("sentry initialization failed", "error", err)
		} else {
			logger.Info("sentry initialized", "environment", environment, "release", version)
		}
	} else {
		logger.Info("sentry disabled (SENTRY_DSN not set)")
	}

	// Validate required configuration before proceeding.
	if errs, warnings := validateConfig(); len(errs) > 0 || len(warnings) > 0 {
		for _, w := range warnings {
			logger.Warn("config warning", "env_var", w.envVar, "detail", w.message)
		}
		if len(errs) > 0 {
			for _, e := range errs {
				logger.Error("config error", "env_var", e.envVar, "detail", e.message)
			}
			log.Fatalf("Startup aborted: %d required configuration value(s) missing", len(errs))
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configure dependencies
	var deps api.Deps
	deps.Logger = logger
	deps.SupabaseJWTSecret = os.Getenv("SUPABASE_JWT_SECRET")
	deps.SupabaseJWKSURL = os.Getenv("SUPABASE_JWKS_URL")
	deps.SupabaseURL = strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	deps.SupabaseServiceRoleKey = os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	// Derive JWKS URL from SUPABASE_URL if not explicitly set.
	// Supabase CLI v2+ uses ES256 (asymmetric signing); the JWKS endpoint
	// provides the public key. Legacy CLI v1 and tests use HS256 + JWT secret.
	if deps.SupabaseJWKSURL == "" {
		if deps.SupabaseURL != "" {
			deps.SupabaseJWKSURL = deps.SupabaseURL + "/auth/v1/.well-known/jwks.json"
		}
	}
	if deps.SupabaseJWKSURL != "" {
		deps.JWKSCache = api.NewJWKSCache(deps.SupabaseJWKSURL)
		log.Printf("JWT: using JWKS endpoint %s (ES256 support enabled)", deps.SupabaseJWKSURL)
	} else if deps.SupabaseJWTSecret != "" {
		log.Printf("JWT: using HS256 secret (legacy/test mode)")
	} else {
		log.Printf("Warning: neither SUPABASE_JWKS_URL/SUPABASE_URL nor SUPABASE_JWT_SECRET is set; authentication will fail")
	}
	deps.BillingEnabled = os.Getenv("BILLING_ENABLED") == "true"
	if deps.BillingEnabled {
		log.Println("Billing: enabled (new users get free plan, Stripe/metering active)")
	} else {
		log.Println("Billing: disabled (all users get unlimited plan, Stripe/metering skipped)")
	}
	deps.DevMode = os.Getenv("MODE") == "development"
	if !deps.DevMode {
		deps.RateLimiter = api.NewRateLimiter(api.DefaultRateLimiterConfig())
		deps.AgentRateLimiter = api.NewRateLimiter(api.DefaultAgentRateLimiterConfig())
		deps.TrustedProxyHeader = os.Getenv("TRUSTED_PROXY_HEADER")
		if deps.TrustedProxyHeader == "" {
			deps.TrustedProxyHeader = "Fly-Client-IP"
		}
		log.Printf("Rate limiting: enabled (per-IP + per-agent + global, proxy header: %s)", deps.TrustedProxyHeader)
	} else {
		log.Println("Rate limiting: disabled (development mode)")
	}
	deps.BaseURL = os.Getenv("BASE_URL")
	deps.InviteHMACKey = os.Getenv("INVITE_HMAC_KEY")

	// Generate ECDSA P-256 key pair for signing action tokens (ES256).
	// The key is ephemeral — tokens are short-lived (≤5 min) so key persistence
	// is not required. A future iteration may load a stable key from env/secrets.
	actionKey, err := ecdsa.GenerateKey(elliptic.P256(), crand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate action token signing key: %v", err)
	}
	deps.ActionTokenSigningKey = actionKey
	deps.ActionTokenKeyID = "action-key-1"
	log.Println("Action token signing: ES256 key generated")

	// Initialize SSE broker for real-time approval notifications.
	deps.ApprovalEvents = api.NewApprovalEventBroker()
	log.Println("Approval events: SSE broker initialized")
	if deps.BaseURL != "" {
		if u, err := url.Parse(deps.BaseURL); err != nil || u.Scheme == "" || u.Host == "" {
			log.Printf("Warning: BASE_URL %q is invalid or not absolute; invite URLs will not be generated", deps.BaseURL)
		}
	}

	// Connect to Postgres if DATABASE_URL is set
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Run pending migrations
		log.Println("Running database migrations...")
		if err := db.Migrate(ctx, dbURL); err != nil {
			sentry.CaptureException(err)
			sentry.Flush(2 * time.Second)
			log.Fatalf("Failed to run migrations: %v", err)
		}

		pool, err := db.Connect(ctx, dbURL)
		if err != nil {
			sentry.CaptureException(err)
			sentry.Flush(2 * time.Second)
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer pool.Close()

		log.Println("Connected to database")
		deps.DB = pool

		// Ensure all existing users have subscriptions. When billing is disabled,
		// this assigns the unlimited pay_as_you_go plan so enforcement sees no limits.
		// When billing is enabled, unsubscribed users get the free plan.
		subCtx, subCancel := context.WithTimeout(context.Background(), 10*time.Second)
		backfilled, err := db.EnsureAllUsersSubscribed(subCtx, pool, deps.BillingEnabled)
		subCancel()
		if err != nil {
			log.Printf("Warning: failed to backfill subscriptions: %v", err)
		} else if backfilled > 0 {
			log.Printf("Subscriptions: backfilled %d user(s) without subscriptions", backfilled)
		}

		// Initialize credential vault.
		// In production/dev with Supabase, use the real vault extension.
		// The vault extension must be enabled in the database.
		deps.Vault = vault.NewSupabaseVaultStore()
		log.Println("Credential vault: using Supabase Vault (encrypted storage)")
	} else {
		log.Println("DATABASE_URL not set, running without database")
	}

	// Initialize notification dispatcher.
	// Channel senders are built from environment variables; each channel
	// issue (#275 Email, #276 Web Push, #277 SMS) adds its own env vars
	// and sender construction to notify.Config.BuildSenders().
	notifyCfg := notify.LoadConfig()
	senders := notifyCfg.BuildSenders()

	// #276 — Web Push (VAPID): Initialize VAPID keys and register sender.
	// VAPID keys require the database for auto-generation + persistence,
	// so this is wired here rather than in BuildSenders().
	if deps.DB != nil {
		vapidCtx, vapidCancel := context.WithTimeout(context.Background(), 10*time.Second)
		vapidKeys, err := webpush.InitVAPIDKeys(vapidCtx, deps.DB, deps.DevMode)
		vapidCancel()
		if err != nil {
			if deps.DevMode {
				log.Printf("Warning: failed to initialize VAPID keys: %v", err)
			} else {
				log.Fatalf("Failed to initialize VAPID keys: %v", err)
			}
		} else if vapidKeys != nil {
			deps.VAPIDPublicKey = vapidKeys.PublicKey
			subject := strings.TrimSpace(notifyCfg.VAPIDSubject)
			if subject == "" {
				if deps.DevMode {
					subject = "mailto:admin@example.com"
					log.Println("Web Push: VAPID_SUBJECT not set, using default mailto:admin@example.com (development mode only)")
				} else {
					log.Fatalf("Web Push: VAPID_SUBJECT is required in production (e.g. mailto:admin@mycompany.com or https://example.com/contact)")
				}
			}
			senders = append(senders, webpush.New(vapidKeys, subject, deps.DB))
		}
	}

	notify.LogChannelSummary(senders)
	if deps.DB != nil && len(senders) > 0 {
		deps.Notifier = notify.NewDispatcher(senders, &notify.DBPreferenceChecker{DB: deps.DB})
	} else if len(senders) > 0 {
		deps.Notifier = notify.NewDispatcher(senders, nil)
	}
	// deps.Notifier is nil when no senders are configured — Dispatch is a no-op.

	// Initialize connector registry.
	registry := connectors.NewRegistry()
	registry.Register(ghconnector.New())
	registry.Register(slack.New())

	// Auto-seed built-in connectors from their manifests.
	if deps.DB != nil {
		seedRegisteredConnectors(registry, deps.DB)
	}

	// Load external connectors from CONNECTORS_DIR (or ~/.permission_slip/connectors/).
	loadExternalConnectors(registry, deps.DB)

	deps.Connectors = registry
	log.Printf("Connector registry: %d connector(s) registered", len(registry.IDs()))

	// Parse allowed CORS origins from a comma-separated list.
	// When empty, the middleware falls back to "same-origin only" mode
	// (derives the server's own origin from Host + TLS state).
	if raw := os.Getenv("ALLOWED_ORIGINS"); raw != "" {
		for _, o := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				deps.AllowedOrigins = append(deps.AllowedOrigins, trimmed)
			}
		}
		// De-duplicate so callers don't need to worry about repeated entries.
		slices.Sort(deps.AllowedOrigins)
		deps.AllowedOrigins = slices.Compact(deps.AllowedOrigins)
		log.Printf("CORS: allowing origins %v", deps.AllowedOrigins)
	} else {
		log.Println("CORS: no ALLOWED_ORIGINS set; same-origin only mode (cross-origin requests will be blocked)")
	}

	// Validate code-registered connectors against database entries.
	if deps.DB != nil {
		validateConnectorRegistry(registry, deps.DB)
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/health", handleHealth(deps.DB))
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", api.NewRouter(&deps)))

	// Invite endpoint lives outside /api/v1/ — it's a user-facing onboarding
	// URL (e.g., https://app.permissionslip.dev/invite/PS-XXXX-XXXX), not a
	// versioned API resource.
	mux.Handle("/invite/", api.InviteHandler(&deps))

	// In production, serve the built React app.
	// In development, use Vite's dev server (port 5173) instead.
	if os.Getenv("MODE") != "development" {
		staticFS, err := fs.Sub(distFS, "frontend/dist")
		if err != nil {
			log.Fatal(err)
		}
		mux.Handle("/", spaHandler(staticFS))
	}

	// Wrap the entire mux with CORS enforcement. This runs before any route
	// handler, including the health check and SPA handler.
	handler := api.CORSMiddleware(deps.AllowedOrigins)(mux)

	// Wrap all routes with security headers (outermost layer).
	// Include the Supabase URL in CSP connect-src so the frontend can reach
	// the auth/API endpoints in production. Sentry's ingest domain is always
	// allowed in connect-src; the optional SENTRY_CSP_ENDPOINT enables
	// report-uri so CSP violations show up as Sentry events.
	sentryCSPEndpoint := os.Getenv("SENTRY_CSP_ENDPOINT")
	var extraConnectSrc []string
	if rawSupabaseURL := strings.TrimSpace(os.Getenv("SUPABASE_URL")); rawSupabaseURL != "" {
		parsed, err := url.Parse(rawSupabaseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			log.Printf("Warning: SUPABASE_URL %q is not a valid URL; skipping CSP connect-src entry", rawSupabaseURL)
		} else {
			extraConnectSrc = append(extraConnectSrc, parsed.Scheme+"://"+parsed.Host)
		}
	}
	// PostHog product analytics — allow the frontend to send events to the
	// PostHog API host. Only added when POSTHOG_HOST is set.
	if posthogHost := strings.TrimSpace(os.Getenv("POSTHOG_HOST")); posthogHost != "" {
		parsed, err := url.Parse(posthogHost)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			log.Printf("Warning: POSTHOG_HOST %q is not a valid URL; skipping CSP connect-src entry", posthogHost)
		} else {
			extraConnectSrc = append(extraConnectSrc, parsed.Scheme+"://"+parsed.Host)
		}
	}
	handler = api.SecurityHeadersMiddleware(sentryCSPEndpoint, extraConnectSrc...)(handler)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
		// ReadHeaderTimeout limits the time allowed to read request headers,
		// defending against Slowloris-style attacks that hold connections open
		// by sending headers very slowly. We intentionally omit ReadTimeout and
		// WriteTimeout because they would break SSE/streaming endpoints.
		ReadHeaderTimeout: 10 * time.Second,
		// IdleTimeout controls how long keep-alive connections remain open
		// between requests. The default (no limit) could allow attackers to
		// exhaust file descriptors. 120s matches common reverse-proxy defaults.
		IdleTimeout: 120 * time.Second,
	}

	// Start server in a goroutine so we can listen for shutdown signals.
	// Errors are sent back to main via srvErr so we don't call log.Fatalf
	// inside the goroutine (which would bypass deferred cleanups like pool.Close).
	srvErr := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", ":"+port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
	}()

	// Block until SIGINT/SIGTERM is received or the server fails to start.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	var serverFailed bool
	select {
	case sig := <-quit:
		logger.Info("shutdown initiated", "signal", sig.String())
	case err := <-srvErr:
		logger.Error("server failed, shutting down", "error", err)
		serverFailed = true
	}

	// Allow up to 30 seconds for in-flight requests to complete.
	shutdownTimeout := 30 * time.Second
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			logger.Warn("invalid SHUTDOWN_TIMEOUT value, using default", "value", v)
		} else {
			shutdownTimeout = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown timed out, forcing close", "error", err)
		srv.Close()
	} else {
		logger.Info("server stopped gracefully")
	}

	// Flush buffered Sentry events before the process exits.
	sentry.Flush(2 * time.Second)

	// Exit non-zero when the server failed (e.g., "address already in use")
	// so process supervisors detect the failure. os.Exit bypasses remaining
	// defers, but at this point only the DB pool close and context cancel are
	// pending — the OS reclaims those resources on process exit.
	if serverFailed {
		os.Exit(1)
	}
}

// loadExternalConnectors scans the default connectors directory for
// subprocess-based connectors, registers them in the registry, and auto-seeds
// their DB rows from manifests so no manual migrations or seed steps are needed.
func loadExternalConnectors(registry *connectors.Registry, d db.DBTX) {
	dir := connectors.DefaultConnectorsDir()
	if dir == "" {
		return
	}

	externals, err := connectors.LoadExternalConnectors(dir)
	if err != nil {
		log.Printf("Warning: failed to load external connectors: %v", err)
		return
	}

	for _, ext := range externals {
		// Refuse to let an external connector shadow a built-in.
		// A malicious manifest with id "github" would otherwise replace the
		// real GitHub connector and receive users' decrypted credentials.
		if _, exists := registry.Get(ext.ID()); exists {
			log.Printf("Warning: external connector %q conflicts with an already-registered connector — skipping", ext.ID())
			continue
		}

		registry.Register(ext)

		// Auto-seed DB rows from the manifest if we have a database connection.
		if d != nil {
			if err := seedConnectorFromManifest(ext.Manifest(), d); err != nil {
				log.Printf("Warning: failed to seed DB for external connector %q: %v", ext.ID(), err)
			}
		}
	}

	// NOTE: We intentionally do NOT delete DB rows for external connectors that
	// are no longer on disk. Deleting would cascade-delete action_configurations
	// that reference those connector actions, breaking agents. The
	// validateConnectorRegistry function logs warnings for this drift instead,
	// and cleanup is left to operators (or a future admin API).

	if len(externals) > 0 {
		log.Printf("External connectors: %d loaded from %s", len(externals), dir)
	}
}

// seedRegisteredConnectors upserts DB rows for all connectors that implement
// ManifestProvider (built-in connectors). This replaces manual seed.go files —
// built-in connectors now follow the same manifest-first pattern as externals.
func seedRegisteredConnectors(registry *connectors.Registry, d db.DBTX) {
	for _, id := range registry.IDs() {
		conn, _ := registry.Get(id)
		mp, ok := conn.(connectors.ManifestProvider)
		if !ok {
			continue
		}
		if err := seedConnectorFromManifest(mp.Manifest(), d); err != nil {
			log.Printf("Warning: failed to seed DB for connector %q: %v", id, err)
		}
	}
}

// seedConnectorFromManifest upserts connector, action, credential, and
// template rows from a connector manifest. Used for both built-in and
// external connectors on server startup.
func seedConnectorFromManifest(manifest *connectors.ConnectorManifest, d db.DBTX) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return db.UpsertConnectorFromManifest(ctx, d, manifest.ToDBManifest())
}

// validateConnectorRegistry logs warnings for mismatches between code-registered
// connectors and database connector entries. This helps catch data/code drift
// during the transition period as connectors are added.
func validateConnectorRegistry(registry *connectors.Registry, d db.DBTX) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbIDs, err := db.ListConnectorIDs(ctx, d)
	if err != nil {
		log.Printf("Warning: failed to list connectors from database for validation: %v", err)
		return
	}

	codeIDs := registry.IDs()

	dbSet := make(map[string]bool, len(dbIDs))
	for _, id := range dbIDs {
		dbSet[id] = true
	}
	codeSet := make(map[string]bool, len(codeIDs))
	for _, id := range codeIDs {
		codeSet[id] = true
	}

	for _, id := range codeIDs {
		if !dbSet[id] {
			log.Printf("Warning: connector %q is registered in code but has no database entry", id)
		}
	}
	for _, id := range dbIDs {
		if !codeSet[id] {
			log.Printf("Warning: connector %q exists in database but has no code registration", id)
		}
	}
}
