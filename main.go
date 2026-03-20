package main

import (
	"context"
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
	_ "github.com/supersuit-tech/permission-slip-web/connectors/all"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/providers"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/notify"
	_ "github.com/supersuit-tech/permission-slip-web/notify/all"
	poauth "github.com/supersuit-tech/permission-slip-web/oauth"
	_ "github.com/supersuit-tech/permission-slip-web/oauth/providers"
	pstripe "github.com/supersuit-tech/permission-slip-web/stripe"
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
	// Emit structured logs for aggregation pipelines AND plain-text emoji
	// summaries so operators can spot issues at a glance in terminal output.
	if errs, warnings := validateConfig(); len(errs) > 0 || len(warnings) > 0 {
		for _, w := range warnings {
			logger.Warn("config warning", "env_var", w.envVar, "detail", w.message)
		}
		if len(warnings) > 0 {
			logger.Warn("⚠️ configuration warnings", "count", len(warnings))
		}
		if len(errs) > 0 {
			for _, e := range errs {
				logger.Error("config error", "env_var", e.envVar, "detail", e.message)
			}
			log.Fatalf("🛑 Startup aborted: %d required configuration value(s) missing", len(errs))
		} else {
			logger.Info("✅ configuration valid — no fatal errors (warnings above are non-blocking)")
		}
	} else {
		logger.Info("✅ all configuration checks passed")
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

		stripeKey := os.Getenv("STRIPE_SECRET_KEY")
		webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
		priceID := os.Getenv("STRIPE_PRICE_ID_REQUEST")

		// Initialize Stripe client when billing is enabled and keys are configured.
		if stripeKey != "" {
			deps.Stripe = pstripe.New(pstripe.Config{
				SecretKey:      stripeKey,
				WebhookSecret:  webhookSecret,
				PriceIDRequest: priceID,
			})
			log.Println("Stripe: client initialized")
			// Fetch and cache the per-request price from Stripe at startup.
			deps.Stripe.FetchRequestPrice()
			if webhookSecret == "" {
				log.Println("Warning: STRIPE_WEBHOOK_SECRET not set — webhook signature verification will reject all requests")
			}
		} else {
			log.Println("Stripe: STRIPE_SECRET_KEY not set, Stripe API calls will be unavailable")
		}
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

	// Initialize SSE broker for real-time approval notifications.
	deps.ApprovalEvents = api.NewApprovalEventBroker()
	log.Println("Approval events: SSE broker initialized")
	if deps.BaseURL != "" {
		if u, err := url.Parse(deps.BaseURL); err != nil || u.Scheme == "" || u.Host == "" {
			log.Printf("Warning: BASE_URL %q is invalid or not absolute; invite URLs will not be generated", deps.BaseURL)
		}
	}

	// Create a cancellable context for background goroutines (e.g. audit purge).
	// Cancelled in the shutdown path so goroutines stop cleanly on SIGTERM.
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	var auditPurgeDone <-chan struct{}
	var bgJobDone []struct {
		name string
		done <-chan struct{}
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

		// Use least-privilege app role for runtime queries, falling back to
		// the superuser DATABASE_URL for backward compatibility.
		appURL := os.Getenv("DATABASE_URL_APP")
		if appURL == "" {
			appURL = dbURL
		}

		pool, err := db.Connect(ctx, appURL)
		if err != nil {
			sentry.CaptureException(err)
			sentry.Flush(2 * time.Second)
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer pool.Close()

		log.Println("Connected to database")
		deps.DB = pool

		// Start background audit log purge.
		auditPurgeDone = startAuditPurge(bgCtx, pool, logger)

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
	// Each channel registers itself via init() in its own package (imported via
	// notify/all). BuildSenders calls all registered factories with the runtime
	// context so each channel can inspect its required config and dependencies.
	notifyCfg := notify.LoadConfig()
	senders := notify.BuildSenders(context.Background(), notify.BuildContext{
		DB:      deps.DB,
		Config:  notifyCfg,
		DevMode: deps.DevMode,
		OnVAPIDPublicKey: func(key string) {
			deps.VAPIDPublicKey = key
		},
	})

	notify.LogChannelSummary(senders)

	// SMS is available when a sender named "sms" was built and the server
	// operator hasn't explicitly hidden it (e.g. on app.permissionslip.dev).
	smsConfigured := false
	for _, s := range senders {
		if s.Name() == "sms" {
			smsConfigured = true
			break
		}
	}
	deps.SMSEnabled = smsConfigured && !strings.EqualFold(os.Getenv("SMS_NOTIFICATIONS_HIDDEN"), "true")
	if deps.DB != nil && len(senders) > 0 {
		deps.Notifier = notify.NewDispatcher(senders, &notify.DBPreferenceChecker{DB: deps.DB})
	} else if len(senders) > 0 {
		deps.Notifier = notify.NewDispatcher(senders, nil)
	}
	// deps.Notifier is nil when no senders are configured — Dispatch is a no-op.

	// Initialize connector registry from self-registered built-in connectors.
	// Each connector package registers itself via init() + connectors.RegisterBuiltIn().
	// The blank import of connectors/all triggers all init() functions.
	registry := connectors.NewRegistry()
	for _, c := range connectors.BuiltInConnectors() {
		registry.Register(c)
	}

	// Auto-seed built-in connectors from their manifests.
	if deps.DB != nil {
		seedRegisteredConnectors(registry, deps.DB)
	}

	// Load external connectors from CONNECTORS_DIR (or ~/.permission_slip/connectors/).
	loadExternalConnectors(registry, deps.DB)

	deps.Connectors = registry

	// Initialize Slack event infrastructure.
	// SLACK_SIGNING_SECRET is the signing secret from the Slack app's Basic
	// Information page — used for HMAC-SHA256 verification of Events API webhooks.
	deps.SlackSigningSecret = os.Getenv("SLACK_SIGNING_SECRET")
	deps.EventBroker = connectors.NewEventBroker()
	// Register the message.im event handler. Currently logs structured event
	// data; extend to trigger workflows when DM automation is implemented.
	deps.EventBroker.Subscribe("message.im", connectors.EventHandlerFunc(
		func(ctx context.Context, event *connectors.Event) error {
			return handleIMMessage(ctx, logger, event)
		},
	))
	if deps.SlackSigningSecret != "" {
		log.Println("Slack events: signing secret configured, Events API webhook enabled")
	} else {
		log.Println("Slack events: SLACK_SIGNING_SECRET not set, Events API webhook will reject requests")
	}

	// Initialize OAuth provider registry with built-in providers (Google, Microsoft)
	// and merge in any providers declared by connector manifests.
	oauthRegistry := poauth.NewRegistryWithBuiltIns()
	registerManifestOAuthProviders(oauthRegistry, registry)
	deps.OAuthProviders = oauthRegistry
	deps.OAuthRedirectBaseURL = os.Getenv("OAUTH_REDIRECT_BASE_URL")
	deps.OAuthStateSecret = os.Getenv("OAUTH_STATE_SECRET")
	log.Printf("OAuth provider registry: %d provider(s) registered", oauthRegistry.Len())
	for _, p := range oauthRegistry.List() {
		if p.HasClientCredentials() {
			log.Printf("  %s: configured (client credentials set)", p.ID)
		} else {
			log.Printf("  %s: registered (no client credentials — BYOA required)", p.ID)
		}
	}

	// Load user BYOA configs from the database and merge into the registry
	// so BYOA credentials survive server restarts.
	if deps.DB != nil && deps.Vault != nil {
		loadBYOAProviderConfigs(oauthRegistry, deps.DB, deps.Vault)
	}

	// Start all registered background jobs.
	for _, job := range backgroundJobs {
		if done := job.Start(bgCtx, &deps, logger); done != nil {
			bgJobDone = append(bgJobDone, struct {
				name string
				done <-chan struct{}
			}{name: job.Name, done: done})
		}
	}

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

	// Stripe webhook endpoint lives outside /api/v1/ — it must bypass auth
	// and rate-limiting middleware. Stripe verifies requests via signature
	// (Stripe-Signature header), not Bearer tokens.
	api.RegisterBillingWebhookRoutes(mux, &deps)

	// Slack Events API webhook endpoint lives outside /api/v1/ — it must
	// bypass auth middleware. Slack authenticates via HMAC-SHA256 signature
	// (X-Slack-Signature header).
	api.RegisterSlackEventRoutes(mux, &deps)

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
	var extraScriptSrc []string
	// PostHog product analytics — allow the frontend to send events and load
	// SDK assets (config.js, toolbar, surveys) from the PostHog proxy host.
	// Added to both connect-src (event ingestion) and script-src (asset loading).
	if posthogHost := strings.TrimSpace(os.Getenv("POSTHOG_HOST")); posthogHost != "" {
		parsed, err := url.Parse(posthogHost)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			log.Printf("Warning: POSTHOG_HOST %q is not a valid URL; skipping CSP entries", posthogHost)
		} else {
			origin := parsed.Scheme + "://" + parsed.Host
			extraConnectSrc = append(extraConnectSrc, origin)
			extraScriptSrc = append(extraScriptSrc, origin)
		}
	}
	// Cloudflare Web Analytics — when CLOUDFLARE_INSIGHTS is "true", allow the
	// auto-injected beacon.min.js script and its data reporting endpoint.
	if strings.EqualFold(strings.TrimSpace(os.Getenv("CLOUDFLARE_INSIGHTS")), "true") {
		extraScriptSrc = append(extraScriptSrc, "https://static.cloudflareinsights.com")
		extraConnectSrc = append(extraConnectSrc, "https://cloudflareinsights.com")
	}
	handler = api.SecurityHeadersMiddleware(sentryCSPEndpoint, extraConnectSrc, extraScriptSrc)(handler)

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

	// Stop background goroutines and wait for them to exit (up to 5s)
	// before closing the DB pool or flushing Sentry.
	bgCancel()
	if auditPurgeDone != nil {
		select {
		case <-auditPurgeDone:
		case <-time.After(5 * time.Second):
			logger.Warn("audit purge goroutine did not exit in time")
		}
	}
	for _, j := range bgJobDone {
		select {
		case <-j.done:
		case <-time.After(5 * time.Second):
			logger.Warn("background job did not exit in time", "job", j.name)
		}
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

// registerManifestOAuthProviders iterates over all connectors in the registry
// and registers any OAuth providers declared in their manifests. This allows
// external connectors to introduce new OAuth providers (e.g. Salesforce) without
// core code changes.
func registerManifestOAuthProviders(oauthReg *poauth.Registry, connReg *connectors.Registry) {
	for _, id := range connReg.IDs() {
		conn, _ := connReg.Get(id)
		mp, ok := conn.(connectors.ManifestProvider)
		if !ok {
			continue
		}
		manifest := mp.Manifest()
		if len(manifest.OAuthProviders) == 0 {
			continue
		}
		providers := make([]poauth.ManifestProvider, len(manifest.OAuthProviders))
		for i, p := range manifest.OAuthProviders {
			providers[i] = poauth.ManifestProvider{
				ID:              p.ID,
				AuthorizeURL:    p.AuthorizeURL,
				TokenURL:        p.TokenURL,
				Scopes:          p.Scopes,
				AuthorizeParams: p.AuthorizeParams,
			}
		}
		if err := poauth.RegisterFromManifest(oauthReg, providers); err != nil {
			log.Printf("Warning: failed to register OAuth providers from connector %q: %v", id, err)
		}
	}
}

// loadBYOAProviderConfigs reads all user BYOA OAuth provider configs from the
// database and merges their client credentials into the in-memory provider
// registry. This ensures BYOA credentials survive server restarts.
//
// Multi-tenancy note: BYOA credentials are stored per-user in the DB, but the
// in-memory registry is global. The last-loaded BYOA config for a given provider
// wins and becomes the active config for ALL users' OAuth flows on this server.
// This is acceptable for single-tenant deployments.
//
// Only configs whose provider already exists in the registry (from built-in or
// manifest sources) are loaded. Configs referencing unknown providers are logged
// as warnings — they'll become active if the provider is registered later.
func loadBYOAProviderConfigs(oauthReg *poauth.Registry, d db.DBTX, v vault.VaultStore) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	configs, err := db.ListAllOAuthProviderConfigs(ctx, d)
	if err != nil {
		log.Printf("Warning: failed to load BYOA provider configs: %v", err)
		return
	}

	var loaded, skipped int
	for _, cfg := range configs {
		// Only load if the provider exists in the registry.
		if _, ok := oauthReg.Get(cfg.Provider); !ok {
			log.Printf("Warning: BYOA config for unknown provider %q (user %s) — skipping", cfg.Provider, cfg.UserID)
			skipped++
			continue
		}

		clientID, err := v.ReadSecret(ctx, d, cfg.ClientIDVaultID)
		if err != nil {
			log.Printf("Warning: failed to read BYOA client_id from vault for provider %q: %v", cfg.Provider, err)
			skipped++
			continue
		}
		clientSecret, err := v.ReadSecret(ctx, d, cfg.ClientSecretVaultID)
		if err != nil {
			log.Printf("Warning: failed to read BYOA client_secret from vault for provider %q: %v", cfg.Provider, err)
			skipped++
			continue
		}

		if err := oauthReg.Register(poauth.Provider{
			ID:           cfg.Provider,
			ClientID:     string(clientID),
			ClientSecret: string(clientSecret),
			Source:       poauth.SourceBYOA,
		}); err != nil {
			log.Printf("Warning: failed to register BYOA provider %q: %v", cfg.Provider, err)
			skipped++
			continue
		}
		loaded++
	}
	if loaded > 0 || skipped > 0 {
		log.Printf("BYOA provider configs: %d loaded, %d skipped", loaded, skipped)
	}
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
