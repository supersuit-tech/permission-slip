package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	osexec "os/exec"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	ghconnector "github.com/supersuit-tech/permission-slip-web/connectors/github"
	googleconnector "github.com/supersuit-tech/permission-slip-web/connectors/google"
	slackconnector "github.com/supersuit-tech/permission-slip-web/connectors/slack"
	"github.com/supersuit-tech/permission-slip-web/db"
)

// runCmd runs a shell command and returns its combined output.
func runCmd(name string, args ...string) (string, error) {
	out, err := osexec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// Deterministic UUIDs for seed users — easy to reference and query.
const (
	userNew                = "00000000-0000-0000-0000-000000000001"
	userHasAgents          = "00000000-0000-0000-0000-000000000002"
	userHasAccessRequests  = "00000000-0000-0000-0000-000000000003"
	userHasActivity        = "00000000-0000-0000-0000-000000000004"
	userHasEverything      = "00000000-0000-0000-0000-000000000005"
	userHasPendingApproval = "00000000-0000-0000-0000-000000000006"
)

var allUserIDs = []string{
	userNew, userHasAgents, userHasAccessRequests, userHasActivity, userHasEverything,
	userHasPendingApproval,
}

// userEmails maps each seed user UUID to their email address.
var userEmails = map[string]string{
	userNew:                "new@test.local",
	userHasAgents:          "has-registered-agents@test.local",
	userHasAccessRequests:  "has-access-requests@test.local",
	userHasActivity:        "has-recent-activity@test.local",
	userHasEverything:      "has-everything@test.local",
	userHasPendingApproval: "has-pending-approvals@test.local",
}

// Connector ID constants for use in scenario seed data below.
const (
	connectorGitHub  = "github"
	connectorSlack   = "slack"
	connectorGoogle  = "google"
)

// supabaseClient holds config for Supabase Admin API calls.
type supabaseClient struct {
	baseURL        string
	serviceRoleKey string
	httpClient     *http.Client
}

func newSupabaseClient() *supabaseClient {
	baseURL := os.Getenv("SUPABASE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:54321"
	}
	key := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if key == "" {
		// Read the secret key from `supabase status` output at runtime so we
		// don't hard-code a key that differs across Supabase versions/installs.
		// Falls back to the classic local-dev JWT for older Supabase setups.
		out, err := runCmd("supabase", "status")
		if err == nil {
			for _, line := range strings.Split(out, "\n") {
				if strings.Contains(line, "Secret") && strings.Contains(line, "sb_secret_") {
					parts := strings.Fields(line)
					for _, p := range parts {
						if strings.HasPrefix(p, "sb_secret_") {
							key = p
							break
						}
					}
				}
			}
		}
		if key == "" {
			// Classic Supabase local-dev service role JWT (fallback).
			key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hj04zWl196z2-SBc0"
		}
	}
	return &supabaseClient{
		baseURL:        baseURL,
		serviceRoleKey: key,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// createUser calls POST /auth/v1/admin/users to create a real Supabase Auth user.
// Retries up to 3 times on 5xx responses (the local Auth service can be slow).
func (s *supabaseClient) createUser(id, email, password string) {
	body := map[string]any{
		"id":            id,
		"email":         email,
		"password":      password,
		"email_confirm": true,
	}
	data, err := json.Marshal(body)
	if err != nil {
		log.Fatalf("createUser: marshal failed: %v", err)
	}

	apiURL := s.baseURL + "/auth/v1/admin/users"
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(data))
		if err != nil {
			log.Fatalf("createUser: new request failed: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)
		req.Header.Set("apikey", s.serviceRoleKey)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			if attempt < maxAttempts {
				log.Printf("createUser %s: attempt %d failed (%v), retrying...", email, attempt, err)
				time.Sleep(2 * time.Second)
				continue
			}
			log.Fatalf("createUser %s: HTTP request failed after %d attempts: %v", email, maxAttempts, err)
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			resp.Body.Close()
			return
		}
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		resp.Body.Close()
		if resp.StatusCode >= 500 && attempt < maxAttempts {
			log.Printf("createUser %s: attempt %d got status %d, retrying...", email, attempt, resp.StatusCode)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Fatalf("createUser %s: unexpected status %d: %v", email, resp.StatusCode, errBody)
	}
}

// listAllUsers returns the IDs of every user in Supabase Auth.
// Used during cleanup to wipe all auth users for a clean seed.
// Paginates through all pages to handle environments with >1000 users.
func (s *supabaseClient) listAllUsers() []string {
	var ids []string
	page := 1
	for {
		apiURL := fmt.Sprintf("%s/auth/v1/admin/users?per_page=1000&page=%d", s.baseURL, page)
		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			log.Fatalf("listAllUsers: new request failed: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)
		req.Header.Set("apikey", s.serviceRoleKey)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			log.Fatalf("listAllUsers: HTTP request failed: %v", err)
		}

		var result struct {
			Users []struct {
				ID string `json:"id"`
			} `json:"users"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			log.Fatalf("listAllUsers: decode failed: %v", err)
		}
		resp.Body.Close()

		for _, u := range result.Users {
			ids = append(ids, u.ID)
		}
		if len(result.Users) == 0 {
			break
		}
		page++
	}
	return ids
}

// deleteUser calls DELETE /auth/v1/admin/users/{id} to remove a Supabase Auth user.
// A 404 is treated as success (user didn't exist — that's fine for cleanup).
// Retries up to 3 times on 5xx responses.
func (s *supabaseClient) deleteUser(id string) {
	apiURL := s.baseURL + "/auth/v1/admin/users/" + id
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodDelete, apiURL, nil)
		if err != nil {
			log.Fatalf("deleteUser %s: new request failed: %v", id, err)
		}
		req.Header.Set("Authorization", "Bearer "+s.serviceRoleKey)
		req.Header.Set("apikey", s.serviceRoleKey)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			if attempt < maxAttempts {
				log.Printf("deleteUser %s: attempt %d failed (%v), retrying...", id, attempt, err)
				time.Sleep(2 * time.Second)
				continue
			}
			log.Fatalf("deleteUser %s: HTTP request failed after %d attempts: %v", id, maxAttempts, err)
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return
		}
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		resp.Body.Close()
		if resp.StatusCode >= 500 && attempt < maxAttempts {
			log.Printf("deleteUser %s: attempt %d got status %d, retrying...", id, attempt, resp.StatusCode)
			time.Sleep(2 * time.Second)
			continue
		}
		log.Fatalf("deleteUser %s: unexpected status %d: %v", id, resp.StatusCode, errBody)
	}
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	supa := newSupabaseClient()

	// Run migrations first so tables exist.
	log.Println("Running migrations...")
	if err := db.Migrate(ctx, dbURL); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Phase 1a: truncate audit_events before deleting auth users. The
	// audit_events FK on profiles uses ON DELETE RESTRICT, so rows there
	// block the cascade from auth.users → profiles that the Auth Admin API
	// triggers. This must happen outside the transaction so the Auth service
	// sees committed state.
	log.Println("Truncating audit_events (FK dep on profiles)...")
	if _, err := pool.Exec(ctx, "TRUNCATE audit_events CASCADE"); err != nil {
		log.Fatalf("Failed to truncate audit_events: %v", err)
	}

	// Phase 1b: delete all auth users via the Admin API (outside any DB
	// transaction so the Auth service sees committed state immediately).
	log.Println("Cleaning previous seed data (auth users)...")
	cleanAuthUsers(supa)

	// Phase 2: clean DB rows and insert new seed data inside a transaction.
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	log.Println("Cleaning previous seed data (DB rows)...")
	cleanDB(ctx, tx)

	log.Println("Seeding data...")
	seed(ctx, tx, supa)

	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Println("Seed complete!")

	// Print Openclaw agent instructions if TEST_AGENT_PUB_KEY was set.
	if testAgentPubKey := os.Getenv("TEST_AGENT_PUB_KEY"); testAgentPubKey != "" {
		fmt.Println("")
		fmt.Println("================================================================")
		fmt.Println("  OPENCLAW AGENT REGISTERED (DEV MODE)")
		fmt.Println("================================================================")
		fmt.Println("")
		fmt.Println("An Openclaw agent has been registered in the local Permission")
		fmt.Println("Slip dev environment for the following test users:")
		fmt.Println("")
		fmt.Println("  - has-registered-agents@test.local")
		fmt.Println("  - has-recent-activity@test.local")
		fmt.Println("  - has-everything@test.local")
		fmt.Println("  - has-pending-approvals@test.local")
		fmt.Println("")
		fmt.Println("Password for all test users: password123")
		fmt.Println("")
		fmt.Println("API Base URL:  http://localhost:8080")
		fmt.Println("Dev App URL:   http://localhost:5173")
		fmt.Println("")
		fmt.Println("Authentication:")
		fmt.Println("  - POST /api/v1/auth/magic-link  (magic link flow)")
		fmt.Println("  - Or use email/password sign-in at http://localhost:5173")
		fmt.Println("")
		fmt.Println("Agent API Authentication:")
		fmt.Println("  Sign requests using the private key corresponding to TEST_AGENT_PUB_KEY.")
		fmt.Println("  (Use whichever key you set — no specific path is assumed)")
		fmt.Println("")
		fmt.Println("⚠️  DEV MODE ONLY — do not use real credentials.")
		fmt.Println("================================================================")
	}
}

// exec is a helper that fatals on error.
func exec(ctx context.Context, tx db.DBTX, sql string, args ...any) {
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		log.Fatalf("exec failed: %v\nSQL: %s", err, sql)
	}
}

// insertAgent inserts an agent row (without specifying agent_id, which is now
// bigserial) and returns the auto-generated agent_id.
func insertAgent(ctx context.Context, tx db.DBTX, sql string, args ...any) int64 {
	var id int64
	if err := tx.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
		log.Fatalf("insertAgent failed: %v\nSQL: %s", err, sql)
	}
	return id
}

// cleanAuthUsers deletes ALL users from Supabase Auth via the Admin API.
// This ensures manually-created accounts (including those with MFA) are also
// wiped so the seed produces a truly clean slate.
func cleanAuthUsers(supa *supabaseClient) {
	ids := supa.listAllUsers()
	for _, id := range ids {
		supa.deleteUser(id)
	}
}

// cleanDB removes all DB rows inside the active transaction.
// Deletes ALL profiles and auth.users (not just seed users) so manually-created
// accounts are also cleaned up, matching what cleanAuthUsers does via the API.
func cleanDB(ctx context.Context, tx db.DBTX) {
	// audit_events is already truncated in Phase 1a (before auth user cleanup).
	exec(ctx, tx, `DELETE FROM profiles`)
	exec(ctx, tx, `DELETE FROM auth.users`)
	exec(ctx, tx, `DELETE FROM connectors WHERE id = $1`, connectorGitHub)
	exec(ctx, tx, `DELETE FROM connectors WHERE id = $1`, connectorGoogle)
	exec(ctx, tx, `DELETE FROM connectors WHERE id = $1`, connectorSlack)
}

// seedOpenclawAgent inserts an Openclaw agent for a user using the real
// TEST_AGENT_PUB_KEY. This allows the Openclaw AI agent to make authenticated
// API calls against the local dev environment.
func seedOpenclawAgent(ctx context.Context, tx db.DBTX, userID, pubKey string) int64 {
	return insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, created_at)
		 VALUES ($1, $2, 'registered', $3, $4, $4)
		 RETURNING agent_id`,
		pubKey, userID,
		`{"name": "Openclaw"}`,
		time.Now())
}

// insertAuditEvent inserts an audit_events row for seed data.
func insertAuditEvent(ctx context.Context, tx db.DBTX, userID string, agentID int64, eventType, outcome, sourceID, sourceType string, agentMeta, action *string, createdAt time.Time) {
	exec(ctx, tx,
		`INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9)`,
		userID, agentID, eventType, outcome, sourceID, sourceType, agentMeta, action, createdAt)
}

// userContactInfo holds optional contact fields for seed profiles.
type userContactInfo struct {
	email string // contact email (not the login email)
	phone string // E.164 format
}

// userContacts maps seed user UUIDs to their contact info.
// Users without an entry have NULL email/phone (no notifications).
var userContacts = map[string]userContactInfo{
	userHasAgents:          {email: "has-registered-agents@example.com", phone: "+15551000002"},
	userHasAccessRequests:  {email: "has-access-requests@example.com"},
	userHasActivity:        {email: "has-recent-activity@example.com", phone: "+15551000004"},
	userHasEverything:      {email: "has-everything@example.com", phone: "+15551000005"},
	userHasPendingApproval: {email: "has-pending-approvals@example.com", phone: "+15551000006"},
}

// insertUser creates a real Supabase Auth user via the Admin API, then inserts
// the corresponding profiles row via SQL (with optional contact fields).
func insertUser(ctx context.Context, tx db.DBTX, supa *supabaseClient, id, email string) {
	supa.createUser(id, email, "password123")
	contact, hasContact := userContacts[id]
	if hasContact && (contact.email != "" || contact.phone != "") {
		var emailPtr, phonePtr *string
		if contact.email != "" {
			emailPtr = &contact.email
		}
		if contact.phone != "" {
			phonePtr = &contact.phone
		}
		exec(ctx, tx, `INSERT INTO profiles (id, username, email, phone) VALUES ($1, $2, $3, $4)`, id, email, emailPtr, phonePtr)
	} else {
		exec(ctx, tx, `INSERT INTO profiles (id, username) VALUES ($1, $2)`, id, email)
	}
}

// seedNotificationPreferences inserts notification preference rows for users
// who have contact info, so the seed data exercises the preferences table.
func seedNotificationPreferences(ctx context.Context, tx db.DBTX) {
	// userHasEverything: all channels enabled
	exec(ctx, tx, `INSERT INTO notification_preferences (user_id, channel, enabled) VALUES ($1, 'email', true)`, userHasEverything)
	exec(ctx, tx, `INSERT INTO notification_preferences (user_id, channel, enabled) VALUES ($1, 'web-push', true)`, userHasEverything)
	exec(ctx, tx, `INSERT INTO notification_preferences (user_id, channel, enabled) VALUES ($1, 'sms', true)`, userHasEverything)

	// userHasActivity: email enabled, sms disabled
	exec(ctx, tx, `INSERT INTO notification_preferences (user_id, channel, enabled) VALUES ($1, 'email', true)`, userHasActivity)
	exec(ctx, tx, `INSERT INTO notification_preferences (user_id, channel, enabled) VALUES ($1, 'sms', false)`, userHasActivity)

	// userHasPendingApproval: web-push disabled (tests default-enabled behavior for unset channels)
	exec(ctx, tx, `INSERT INTO notification_preferences (user_id, channel, enabled) VALUES ($1, 'web-push', false)`, userHasPendingApproval)

	fmt.Println("  ✓ notification_preferences seeded")
}

func seedPushSubscriptions(ctx context.Context, tx db.DBTX) {
	// userHasEverything: has a push subscription (simulates an enrolled browser)
	exec(ctx, tx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES ($1, $2, $3, $4)`,
		userHasEverything,
		"https://fcm.googleapis.com/fcm/send/seed-subscription-1",
		"BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8p8REqnSw",
		"tBHItJI5svbpC7RB5gg6bQ",
	)

	fmt.Println("  \u2713 push_subscriptions seeded")
}

// seedSubscriptions creates a free subscription for every seed user and
// upgrades userHasEverything to pay_as_you_go to exercise both plan tiers.
func seedSubscriptions(ctx context.Context, tx db.DBTX) {
	for _, uid := range allUserIDs {
		plan := "free"
		if uid == userHasEverything {
			plan = "pay_as_you_go"
		}
		exec(ctx, tx,
			`INSERT INTO subscriptions (user_id, plan_id) VALUES ($1, $2)`,
			uid, plan)
	}
	fmt.Println("  ✓ subscriptions seeded")
}

// seedUsagePeriods creates sample usage data for active users.
func seedUsagePeriods(ctx context.Context, tx db.DBTX) {
	periodStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// userHasActivity: moderate usage on free tier
	exec(ctx, tx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count, sms_count) VALUES ($1, $2, $3, $4, $5)`,
		userHasActivity, periodStart, periodEnd, 250, 0)

	// userHasEverything: heavy usage on pay-as-you-go tier
	exec(ctx, tx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count, sms_count) VALUES ($1, $2, $3, $4, $5)`,
		userHasEverything, periodStart, periodEnd, 2500, 12)

	// userHasAgents: near the free-tier limit
	exec(ctx, tx,
		`INSERT INTO usage_periods (user_id, period_start, period_end, request_count, sms_count) VALUES ($1, $2, $3, $4, $5)`,
		userHasAgents, periodStart, periodEnd, 950, 0)

	fmt.Println("  ✓ usage_periods seeded")
}

func seed(ctx context.Context, tx db.DBTX, supa *supabaseClient) {
	seedConnectors(ctx, tx)
	seedUserNew(ctx, tx, supa)
	seedUserHasAgents(ctx, tx, supa)
	seedUserHasAccessRequests(ctx, tx, supa)
	seedUserHasActivity(ctx, tx, supa)
	seedUserHasEverything(ctx, tx, supa)
	seedUserHasPendingApprovals(ctx, tx, supa)
	seedNotificationPreferences(ctx, tx)
	seedPushSubscriptions(ctx, tx)
	seedSubscriptions(ctx, tx)
	seedUsagePeriods(ctx, tx)
	seedAuditEvents(ctx, tx)
}

// seedAuditEvents populates the audit_events table from the seed data that was
// just inserted into approvals and agents. This mirrors the backfill in the
// migration but operates on the freshly-seeded data.
func seedAuditEvents(ctx context.Context, tx db.DBTX) {
	// Backfill from resolved approvals.
	// Only the action type is stored — parameters are redacted from the audit trail.
	exec(ctx, tx, `
		INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
		SELECT
			a.approver_id,
			a.agent_id,
			CASE a.status
				WHEN 'approved' THEN 'approval.approved'
				WHEN 'denied' THEN 'approval.denied'
				WHEN 'cancelled' THEN 'approval.cancelled'
			END,
			a.status,
			a.approval_id,
			'approval',
			ag.metadata,
			jsonb_build_object('type', a.action->>'type'),
			COALESCE(a.approved_at, a.denied_at, a.cancelled_at)
		FROM approvals a
		JOIN agents ag ON ag.agent_id = a.agent_id
		WHERE a.status IN ('approved', 'denied', 'cancelled')
		  AND COALESCE(a.approved_at, a.denied_at, a.cancelled_at) IS NOT NULL`)

	// Backfill from agent registrations.
	exec(ctx, tx, `
		INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
		SELECT
			ag.approver_id,
			ag.agent_id,
			'agent.registered',
			'registered',
			'ar:' || ag.agent_id::text,
			'agent',
			ag.metadata,
			NULL,
			ag.registered_at
		FROM agents ag
		WHERE ag.registered_at IS NOT NULL`)

	// Backfill from pending agent registrations (these will resolve to "expired"
	// at query time if expires_at <= now()).
	exec(ctx, tx, `
		INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
		SELECT
			ag.approver_id,
			ag.agent_id,
			'agent.registered',
			'pending',
			'ri:seed-' || ag.agent_id::text,
			'registration_invite',
			ag.metadata,
			NULL,
			ag.created_at
		FROM agents ag
		WHERE ag.status = 'pending'`)

	// Backfill from agent deactivations.
	exec(ctx, tx, `
		INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
		SELECT
			ag.approver_id,
			ag.agent_id,
			'agent.deactivated',
			'deactivated',
			'ad:' || ag.agent_id::text,
			'agent',
			ag.metadata,
			NULL,
			ag.deactivated_at
		FROM agents ag
		WHERE ag.deactivated_at IS NOT NULL`)

	// Backfill from standing approval executions.
	// Only the action type is stored — parameters and constraints are redacted.
	exec(ctx, tx, `
		INSERT INTO audit_events (user_id, agent_id, event_type, outcome, source_id, source_type, agent_meta, action, created_at)
		SELECT
			sa.user_id,
			sa.agent_id,
			'standing_approval.executed',
			'auto_executed',
			'sae:' || sae.id::text,
			'standing_approval',
			ag.metadata,
			jsonb_build_object('type', sa.action_type),
			sae.executed_at
		FROM standing_approval_executions sae
		JOIN standing_approvals sa ON sa.standing_approval_id = sae.standing_approval_id
		JOIN agents ag ON ag.agent_id = sa.agent_id AND ag.approver_id = sa.user_id`)

	// Count for log output.
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM audit_events`).Scan(&count); err != nil {
		log.Printf("  Warning: could not count audit_events: %v", err)
	} else {
		fmt.Printf("  ✓ Audit events: %d rows backfilled\n", count)
	}
}

// ---------------------------------------------------------------------------
// Connectors (shared reference data)
// ---------------------------------------------------------------------------

// seedConnectors upserts connector reference data from each built-in
// connector's manifest. When adding a new connector, add its manifest here.
func seedConnectors(ctx context.Context, tx db.DBTX) {
	builtins := []struct {
		manifest *connectors.ConnectorManifest
	}{
		{ghconnector.New().Manifest()},
		{googleconnector.New().Manifest()},
		{slackconnector.New().Manifest()},
	}

	for _, b := range builtins {
		if err := b.manifest.Validate(); err != nil {
			log.Fatalf("Built-in connector %q has invalid manifest: %v", b.manifest.ID, err)
		}
		if err := db.UpsertConnectorFromManifest(ctx, tx, b.manifest.ToDBManifest()); err != nil {
			log.Fatalf("Failed to seed connector %q: %v", b.manifest.ID, err)
		}
	}

	fmt.Println("  ✓ Connectors: github, google, slack (with templates)")
}

// ---------------------------------------------------------------------------
// 1. new@test.local — profile only
// ---------------------------------------------------------------------------

func seedUserNew(ctx context.Context, tx db.DBTX, supa *supabaseClient) {
	insertUser(ctx, tx, supa, userNew, "new@test.local")
	fmt.Println("  ✓ new@test.local (profile only)")
}

// ---------------------------------------------------------------------------
// 2. has-registered-agents@test.local — several agents in various states
// ---------------------------------------------------------------------------

func seedUserHasAgents(ctx context.Context, tx db.DBTX, supa *supabaseClient) {
	insertUser(ctx, tx, supa, userHasAgents, "has-registered-agents@test.local")

	now := time.Now()

	// 3 registered agents
	agents := []struct {
		name       string
		registedAt time.Time
	}{
		{"Claude Code", now.Add(-7 * 24 * time.Hour)},
		{"GitHub Bot", now.Add(-14 * 24 * time.Hour)},
		{"Deploy Agent", now.Add(-30 * 24 * time.Hour)},
	}
	var registeredAgentIDs []int64
	for _, a := range agents {
		id := insertAgent(ctx, tx,
			`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, created_at)
			 VALUES ($1, $2, 'registered', $3, $4, $4)
			 RETURNING agent_id`,
			fmt.Sprintf("pk-seed-%s", a.name), userHasAgents,
			fmt.Sprintf(`{"name": "%s"}`, a.name),
			a.registedAt)
		registeredAgentIDs = append(registeredAgentIDs, id)
	}

	// Enable connectors for the first two registered agents
	// Claude Code → GitHub connector
	if len(registeredAgentIDs) >= 1 {
		exec(ctx, tx,
			`INSERT INTO agent_connectors (agent_id, approver_id, connector_id) VALUES ($1, $2, $3)`,
			registeredAgentIDs[0], userHasAgents, connectorGitHub)
	}
	// GitHub Bot → GitHub + Slack connectors
	if len(registeredAgentIDs) >= 2 {
		exec(ctx, tx,
			`INSERT INTO agent_connectors (agent_id, approver_id, connector_id) VALUES ($1, $2, $3)`,
			registeredAgentIDs[1], userHasAgents, connectorGitHub)
		exec(ctx, tx,
			`INSERT INTO agent_connectors (agent_id, approver_id, connector_id) VALUES ($1, $2, $3)`,
			registeredAgentIDs[1], userHasAgents, connectorSlack)
	}

	// 1 pending agent (with confirmation code for dashboard display)
	insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, confirmation_code, expires_at, created_at)
		 VALUES ($1, $2, 'pending', $3, $4, $5, $6)
		 RETURNING agent_id`,
		"pk-seed-pending-setup", userHasAgents,
		`{"name": "New Agent (Pending)"}`,
		"XK7-M9P",
		now.Add(1*time.Hour),
		now.Add(-1*time.Hour))

	// 1 deactivated agent
	insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, deactivated_at, created_at)
		 VALUES ($1, $2, 'deactivated', $3, $4, $5, $6)
		 RETURNING agent_id`,
		"pk-seed-old-retired", userHasAgents,
		`{"name": "Old Retired Bot"}`,
		now.Add(-60*24*time.Hour),
		now.Add(-5*24*time.Hour),
		now.Add(-60*24*time.Hour))

	// Openclaw agent (if TEST_AGENT_PUB_KEY is set)
	if testAgentPubKey := os.Getenv("TEST_AGENT_PUB_KEY"); testAgentPubKey != "" {
		seedOpenclawAgent(ctx, tx, userHasAgents, testAgentPubKey)
	}

	if os.Getenv("TEST_AGENT_PUB_KEY") != "" {
		fmt.Println("  ✓ has-registered-agents@test.local (4 registered (incl. Openclaw), 1 pending, 1 deactivated, 3 agent-connectors)")
	} else {
		fmt.Println("  ✓ has-registered-agents@test.local (3 registered, 1 pending, 1 deactivated, 3 agent-connectors)")
	}
}

// ---------------------------------------------------------------------------
// 3. has-access-requests@test.local — approval requests in various states
// ---------------------------------------------------------------------------

func seedUserHasAccessRequests(ctx context.Context, tx db.DBTX, supa *supabaseClient) {
	insertUser(ctx, tx, supa, userHasAccessRequests, "has-access-requests@test.local")

	now := time.Now()

	// 1 registered agent to own the approvals
	agentID := insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, created_at)
		 VALUES ($1, $2, 'registered', $3, $4, $4)
		 RETURNING agent_id`,
		"pk-seed-requests-bot", userHasAccessRequests,
		`{"name": "Requests Bot"}`,
		now.Add(-10*24*time.Hour))

	// Pending approvals (24h expiry so they survive dev sessions)
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-pending-1", agentID, userHasAccessRequests,
		`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/webapp", "title": "Fix login bug"}}`,
		`{"description": "Create an issue to track the login bug", "urgency": "high"}`,
		now.Add(24*time.Hour),
		now.Add(-10*time.Minute))

	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-pending-2", agentID, userHasAccessRequests,
		`{"type": "slack.send_message", "parameters": {"channel": "#deployments", "message": "Deploying v2.1.0"}}`,
		`{"description": "Notify the team about the upcoming deployment"}`,
		now.Add(24*time.Hour),
		now.Add(-5*time.Minute))

	// Approved (recent)
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, approved_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'approved', $6, $7, $8)`,
		"approval-approved-recent", agentID, userHasAccessRequests,
		`{"type": "github.merge_pr", "parameters": {"repo": "supersuit-tech/webapp", "pr": 42}}`,
		`{"description": "Merge the feature branch for auth improvements"}`,
		now.Add(1*time.Hour),
		now.Add(-2*time.Hour),
		now.Add(-3*time.Hour))

	// Approved (older)
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, approved_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'approved', $6, $7, $8)`,
		"approval-approved-old", agentID, userHasAccessRequests,
		`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/api", "title": "Update rate limits"}}`,
		`{"description": "Track the rate limiting changes needed for v3"}`,
		now.Add(-13*24*time.Hour),
		now.Add(-14*24*time.Hour),
		now.Add(-14*24*time.Hour))

	// Denied
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, denied_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'denied', $6, $7, $8)`,
		"approval-denied-1", agentID, userHasAccessRequests,
		`{"type": "slack.create_channel", "parameters": {"name": "random-test"}}`,
		`{"description": "Create a test channel — denied because it already exists"}`,
		now.Add(-4*24*time.Hour),
		now.Add(-5*24*time.Hour),
		now.Add(-5*24*time.Hour))

	// Cancelled
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, cancelled_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'cancelled', $6, $7, $8)`,
		"approval-cancelled-1", agentID, userHasAccessRequests,
		`{"type": "github.merge_pr", "parameters": {"repo": "supersuit-tech/webapp", "pr": 38}}`,
		`{"description": "Cancelled — PR had merge conflicts"}`,
		now.Add(-1*24*time.Hour),
		now.Add(-2*24*time.Hour),
		now.Add(-2*24*time.Hour))

	fmt.Println("  ✓ has-access-requests@test.local (2 pending, 2 approved, 1 denied, 1 cancelled)")
}

// ---------------------------------------------------------------------------
// 4. has-recent-activity@test.local — active agents with recent requests
// ---------------------------------------------------------------------------

func seedUserHasActivity(ctx context.Context, tx db.DBTX, supa *supabaseClient) {
	insertUser(ctx, tx, supa, userHasActivity, "has-recent-activity@test.local")

	now := time.Now()

	// 2 registered agents with recent last_active_at
	agentCI := insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, last_active_at, created_at)
		 VALUES ($1, $2, 'registered', $3, $4, $5, $4)
		 RETURNING agent_id`,
		"pk-seed-active-ci", userHasActivity,
		`{"name": "CI Pipeline Agent"}`,
		now.Add(-20*24*time.Hour),
		now.Add(-30*time.Minute))

	agentMonitor := insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, last_active_at, created_at)
		 VALUES ($1, $2, 'registered', $3, $4, $5, $4)
		 RETURNING agent_id`,
		"pk-seed-active-monitor", userHasActivity,
		`{"name": "Monitoring Agent"}`,
		now.Add(-15*24*time.Hour),
		now.Add(-2*time.Hour))

	// Recent approvals for CI agent (within last 30 days)
	for i, daysAgo := range []int{0, 1, 3, 5, 7, 14, 21} {
		createdAt := now.Add(-time.Duration(daysAgo) * 24 * time.Hour)
		exec(ctx, tx,
			`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, approved_at, created_at)
			 VALUES ($1, $2, $3, $4, $5, 'approved', $6, $7, $8)`,
			fmt.Sprintf("approval-activity-ci-%d", i),
			agentCI, userHasActivity,
			fmt.Sprintf(`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/webapp", "run": %d}}`, 100+i),
			`{"description": "CI pipeline action"}`,
			createdAt.Add(1*time.Hour),
			createdAt.Add(5*time.Minute),
			createdAt)
	}

	// Recent approvals for monitor agent
	for i, daysAgo := range []int{0, 2, 6, 10} {
		createdAt := now.Add(-time.Duration(daysAgo) * 24 * time.Hour)
		exec(ctx, tx,
			`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, approved_at, created_at)
			 VALUES ($1, $2, $3, $4, $5, 'approved', $6, $7, $8)`,
			fmt.Sprintf("approval-activity-monitor-%d", i),
			agentMonitor, userHasActivity,
			`{"type": "slack.send_message", "parameters": {"channel": "#alerts"}}`,
			`{"description": "Monitoring alert notification"}`,
			createdAt.Add(1*time.Hour),
			createdAt.Add(2*time.Minute),
			createdAt)
	}

	// 1 active standing approval
	exec(ctx, tx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, constraints, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6, $7)`,
		"sa-activity-1", agentCI, userHasActivity,
		"github.create_issue",
		`{"repo": "supersuit-tech/ci-actions"}`,
		now.Add(-7*24*time.Hour),
		now.Add(23*24*time.Hour))

	// Openclaw agent (if TEST_AGENT_PUB_KEY is set)
	if testAgentPubKey := os.Getenv("TEST_AGENT_PUB_KEY"); testAgentPubKey != "" {
		seedOpenclawAgent(ctx, tx, userHasActivity, testAgentPubKey)
	}

	if os.Getenv("TEST_AGENT_PUB_KEY") != "" {
		fmt.Println("  ✓ has-recent-activity@test.local (3 active agents (incl. Openclaw), 11 recent approvals, 1 standing approval)")
	} else {
		fmt.Println("  ✓ has-recent-activity@test.local (2 active agents, 11 recent approvals, 1 standing approval)")
	}
}

// ---------------------------------------------------------------------------
// 5. has-everything@test.local — the full demo account
// ---------------------------------------------------------------------------

func seedUserHasEverything(ctx context.Context, tx db.DBTX, supa *supabaseClient) {
	insertUser(ctx, tx, supa, userHasEverything, "has-everything@test.local")

	now := time.Now()

	// ---------------------------------------------------------------
	// Registered agents (7 total — a mix of recently active and idle)
	// ---------------------------------------------------------------
	type regAgent struct {
		name         string
		registeredAt time.Duration
		lastActiveAt time.Duration // relative to now
	}
	regAgentDefs := []regAgent{
		{"Claude Code", -30 * 24 * time.Hour, -1 * time.Hour},
		{"GitHub Actions Bot", -20 * 24 * time.Hour, -3 * time.Hour},
		{"Slack Notifier", -10 * 24 * time.Hour, -6 * time.Hour},
		{"Jira Sync Agent", -45 * 24 * time.Hour, -12 * time.Hour},
		{"Sentry Error Reporter", -25 * 24 * time.Hour, -2 * time.Hour},
		{"Datadog Monitor", -15 * 24 * time.Hour, -30 * time.Minute},
		{"PagerDuty Escalator", -8 * 24 * time.Hour, -4 * time.Hour},
	}
	// Map name → generated agent_id for referencing in approvals below.
	agentIDs := make(map[string]int64)
	for _, a := range regAgentDefs {
		id := insertAgent(ctx, tx,
			`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, last_active_at, created_at)
			 VALUES ($1, $2, 'registered', $3, $4, $5, $4)
			 RETURNING agent_id`,
			fmt.Sprintf("pk-seed-%s", a.name), userHasEverything,
			fmt.Sprintf(`{"name": "%s"}`, a.name),
			now.Add(a.registeredAt),
			now.Add(a.lastActiveAt))
		agentIDs[a.name] = id
	}

	// 1 pending agent (with confirmation code for dashboard display)
	insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, confirmation_code, expires_at, created_at)
		 VALUES ($1, $2, 'pending', $3, $4, $5, $6)
		 RETURNING agent_id`,
		"pk-seed-everything-pending", userHasEverything,
		`{"name": "New CI Runner (Pending)"}`,
		"AB3-CD5",
		now.Add(1*time.Hour),
		now.Add(-30*time.Minute))

	// 2 deactivated agents
	insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, deactivated_at, created_at)
		 VALUES ($1, $2, 'deactivated', $3, $4, $5, $4)
		 RETURNING agent_id`,
		"pk-seed-everything-retired", userHasEverything,
		`{"name": "Legacy Deploy Bot"}`,
		now.Add(-90*24*time.Hour),
		now.Add(-10*24*time.Hour))

	insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, deactivated_at, created_at)
		 VALUES ($1, $2, 'deactivated', $3, $4, $5, $4)
		 RETURNING agent_id`,
		"pk-seed-everything-old-slack", userHasEverything,
		`{"name": "Old Slack Bot (v1)"}`,
		now.Add(-60*24*time.Hour),
		now.Add(-20*24*time.Hour))

	// Shorthand lookups for the 7 registered agents.
	claude := agentIDs["Claude Code"]
	github := agentIDs["GitHub Actions Bot"]
	slack := agentIDs["Slack Notifier"]
	jira := agentIDs["Jira Sync Agent"]
	sentry := agentIDs["Sentry Error Reporter"]
	datadog := agentIDs["Datadog Monitor"]
	pagerduty := agentIDs["PagerDuty Escalator"]

	// ---------------------------------------------------------------
	// Agent connectors — enable GitHub, Slack, and/or Google for agents
	// ---------------------------------------------------------------
	agentConnectors := []struct {
		agentID     int64
		connectorID string
	}{
		{claude, connectorGitHub},
		{github, connectorGitHub},
		{slack, connectorSlack},
		{sentry, connectorGitHub},
		{datadog, connectorSlack},
		{pagerduty, connectorSlack},
		// Google connector on all 7 pre-seeded registered agents (Openclaw excluded)
		{claude, connectorGoogle},
		{github, connectorGoogle},
		{slack, connectorGoogle},
		{jira, connectorGoogle},
		{sentry, connectorGoogle},
		{datadog, connectorGoogle},
		{pagerduty, connectorGoogle},
	}
	for _, ac := range agentConnectors {
		exec(ctx, tx,
			`INSERT INTO agent_connectors (agent_id, approver_id, connector_id) VALUES ($1, $2, $3)`,
			ac.agentID, userHasEverything, ac.connectorID)
	}

	// ---------------------------------------------------------------
	// OAuth connections — two Google accounts for testing multi-account
	//
	// vault_secret_id values are placeholder UUIDs — they don't reference
	// real vault secrets. Seed data is for UI development; actual vault
	// integration is tested via MockVaultStore in Go tests.
	// ---------------------------------------------------------------
	exec(ctx, tx,
		`INSERT INTO oauth_connections (id, user_id, provider, access_token_vault_id, scopes, status, extra_data)
		 VALUES ($1, $2, $3, $4, $5, 'active', $6)`,
		"oauth-google-work", userHasEverything, "google",
		"00000000-0000-0000-0000-000000000080",
		`{https://www.googleapis.com/auth/gmail.send,https://www.googleapis.com/auth/calendar.events,https://www.googleapis.com/auth/drive}`,
		`{"email": "alice@supersuit.com", "name": "Alice (Work)"}`)

	exec(ctx, tx,
		`INSERT INTO oauth_connections (id, user_id, provider, access_token_vault_id, scopes, status, extra_data)
		 VALUES ($1, $2, $3, $4, $5, 'active', $6)`,
		"oauth-google-personal", userHasEverything, "google",
		"00000000-0000-0000-0000-000000000081",
		`{https://www.googleapis.com/auth/gmail.send,https://www.googleapis.com/auth/calendar.events,https://www.googleapis.com/auth/drive}`,
		`{"email": "alice.personal@gmail.com", "name": "Alice (Personal)"}`)

	// ---------------------------------------------------------------
	// Agent-connector credentials — bind one Google OAuth per agent
	// ---------------------------------------------------------------
	// Alternate between the two Google accounts so both appear in use.
	googleOAuthBindings := []struct {
		id        string
		agentID   int64
		oauthConn string
	}{
		{"acc-claude-google", claude, "oauth-google-work"},
		{"acc-github-google", github, "oauth-google-personal"},
		{"acc-slack-google", slack, "oauth-google-work"},
		{"acc-jira-google", jira, "oauth-google-personal"},
		{"acc-sentry-google", sentry, "oauth-google-work"},
		{"acc-datadog-google", datadog, "oauth-google-personal"},
		{"acc-pagerduty-google", pagerduty, "oauth-google-work"},
	}
	for _, b := range googleOAuthBindings {
		exec(ctx, tx,
			`INSERT INTO agent_connector_credentials (id, agent_id, connector_id, approver_id, oauth_connection_id)
			 VALUES ($1, $2, $3, $4, $5)`,
			b.id, b.agentID, connectorGoogle, userHasEverything, b.oauthConn)
	}

	// ---------------------------------------------------------------
	// Approvals — pending (3, 24h expiry so they survive dev sessions)
	// ---------------------------------------------------------------
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-everything-pending-1", claude, userHasEverything,
		`{"type": "github.merge_pr", "parameters": {"repo": "supersuit-tech/webapp", "pr": 99}}`,
		`{"description": "Merge the refactoring PR"}`,
		now.Add(24*time.Hour),
		now.Add(-15*time.Minute))

	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-everything-pending-2", sentry, userHasEverything,
		`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/api", "title": "Unhandled TypeError in /users endpoint"}}`,
		`{"description": "Auto-create issue from Sentry alert #4821"}`,
		now.Add(24*time.Hour),
		now.Add(-5*time.Minute))

	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-everything-pending-3", datadog, userHasEverything,
		`{"type": "slack.send_message", "parameters": {"channel": "#incidents", "message": "CPU usage above 90% on prod-web-3"}}`,
		`{"description": "Alert the on-call team about high CPU"}`,
		now.Add(24*time.Hour),
		now.Add(-2*time.Minute))

	// ---------------------------------------------------------------
	// Approvals — approved (across multiple agents, recent)
	// ---------------------------------------------------------------
	approvedApprovals := []struct {
		id      string
		agentID int64
		action  string
		context string
		age     time.Duration // how long ago it was created
	}{
		{"approval-everything-approved-1", github,
			`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/webapp", "title": "Upgrade dependencies"}}`,
			`{"description": "Track dependency upgrades for Q1"}`,
			1 * 24 * time.Hour},
		{"approval-everything-approved-2", claude,
			`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/api", "title": "Add caching layer"}}`,
			`{"description": "Performance improvement for API responses"}`,
			5 * 24 * time.Hour},
		{"approval-everything-approved-3", slack,
			`{"type": "slack.send_message", "parameters": {"channel": "#releases", "message": "v2.3.0 deployed to production"}}`,
			`{"description": "Release notification"}`,
			2 * 24 * time.Hour},
		{"approval-everything-approved-4", jira,
			`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/webapp", "title": "Sync JIRA-1234 status"}}`,
			`{"description": "Bi-directional Jira sync"}`,
			3 * 24 * time.Hour},
		{"approval-everything-approved-5", pagerduty,
			`{"type": "slack.send_message", "parameters": {"channel": "#oncall", "message": "Incident resolved — DB connection pool recovered"}}`,
			`{"description": "Auto-resolve notification from PagerDuty"}`,
			12 * time.Hour},
	}
	for _, a := range approvedApprovals {
		createdAt := now.Add(-a.age)
		exec(ctx, tx,
			`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, approved_at, created_at)
			 VALUES ($1, $2, $3, $4, $5, 'approved', $6, $7, $8)`,
			a.id, a.agentID, userHasEverything,
			a.action, a.context,
			createdAt.Add(1*time.Hour),
			createdAt.Add(5*time.Minute),
			createdAt)
	}

	// ---------------------------------------------------------------
	// Approvals — denied (2)
	// ---------------------------------------------------------------
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, denied_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'denied', $6, $7, $8)`,
		"approval-everything-denied-1", slack, userHasEverything,
		`{"type": "slack.create_channel", "parameters": {"name": "spam-channel"}}`,
		`{"description": "Denied — inappropriate channel name"}`,
		now.Add(-2*24*time.Hour),
		now.Add(-3*24*time.Hour),
		now.Add(-3*24*time.Hour))

	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, denied_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'denied', $6, $7, $8)`,
		"approval-everything-denied-2", claude, userHasEverything,
		`{"type": "github.merge_pr", "parameters": {"repo": "supersuit-tech/webapp", "pr": 87}}`,
		`{"description": "Denied — tests still failing on this PR"}`,
		now.Add(-5*24*time.Hour),
		now.Add(-6*24*time.Hour),
		now.Add(-6*24*time.Hour))

	// ---------------------------------------------------------------
	// Approvals — cancelled (2)
	// ---------------------------------------------------------------
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, cancelled_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'cancelled', $6, $7, $8)`,
		"approval-everything-cancelled-1", github, userHasEverything,
		`{"type": "github.merge_pr", "parameters": {"repo": "supersuit-tech/api", "pr": 55}}`,
		`{"description": "Cancelled — waiting for code review"}`,
		now.Add(-6*24*time.Hour),
		now.Add(-7*24*time.Hour),
		now.Add(-7*24*time.Hour))

	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, cancelled_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'cancelled', $6, $7, $8)`,
		"approval-everything-cancelled-2", jira, userHasEverything,
		`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/webapp", "title": "Duplicate task"}}`,
		`{"description": "Cancelled — issue already existed"}`,
		now.Add(-4*24*time.Hour),
		now.Add(-4*24*time.Hour),
		now.Add(-5*24*time.Hour))

	// ---------------------------------------------------------------
	// Historical approvals (spread across agents for request_count_30d)
	// ---------------------------------------------------------------
	histApprovals := []struct {
		suffix  string
		agentID int64
		daysAgo int
	}{
		{"claude-1", claude, 2},
		{"claude-2", claude, 8},
		{"claude-3", claude, 15},
		{"claude-4", claude, 22},
		{"claude-5", claude, 28},
		{"github-1", github, 1},
		{"github-2", github, 9},
		{"github-3", github, 18},
		{"sentry-1", sentry, 0},
		{"sentry-2", sentry, 4},
		{"sentry-3", sentry, 11},
		{"datadog-1", datadog, 0},
		{"datadog-2", datadog, 3},
		{"datadog-3", datadog, 7},
		{"datadog-4", datadog, 14},
		{"pagerduty-1", pagerduty, 1},
		{"pagerduty-2", pagerduty, 6},
	}
	for _, h := range histApprovals {
		createdAt := now.Add(-time.Duration(h.daysAgo) * 24 * time.Hour)
		exec(ctx, tx,
			`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, approved_at, created_at)
			 VALUES ($1, $2, $3, $4, $5, 'approved', $6, $7, $8)`,
			"approval-everything-hist-"+h.suffix,
			h.agentID, userHasEverything,
			`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/webapp"}}`,
			`{"description": "Historical approval"}`,
			createdAt.Add(1*time.Hour),
			createdAt.Add(5*time.Minute),
			createdAt)
	}

	// ---------------------------------------------------------------
	// Credentials (2)
	//
	// vault_secret_id values are placeholder UUIDs — they don't reference
	// real vault secrets. Seed data is for UI development; actual vault
	// integration is tested via MockVaultStore in Go tests and via
	// Supabase Vault in local dev integration tests.
	// ---------------------------------------------------------------
	exec(ctx, tx,
		`INSERT INTO credentials (id, user_id, service, label, vault_secret_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		"cred-github-token", userHasEverything, "github", "Personal Access Token",
		"00000000-0000-0000-0000-000000000099")

	exec(ctx, tx,
		`INSERT INTO credentials (id, user_id, service, label, vault_secret_id)
		 VALUES ($1, $2, $3, $4, $5)`,
		"cred-slack-oauth", userHasEverything, "slack", "Workspace OAuth Token",
		"00000000-0000-0000-0000-000000000098")

	// ---------------------------------------------------------------
	// Standing approvals (2 active, 1 expired)
	// ---------------------------------------------------------------
	exec(ctx, tx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, max_executions, constraints, source_action_configuration_id, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6, $7, $8, $9)`,
		"sa-everything-1", claude, userHasEverything,
		"github.create_issue", 100,
		`{"repo": "supersuit-tech/permission-slip", "title": "*"}`,
		"ac-claude-create-issue",
		now.Add(-7*24*time.Hour),
		now.Add(23*24*time.Hour))

	exec(ctx, tx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, constraints, starts_at, expires_at)
		 VALUES ($1, $2, $3, $4, 'active', $5, $6, $7)`,
		"sa-everything-2", slack, userHasEverything,
		"slack.send_message",
		`{"channel": "#engineering"}`,
		now.Add(-3*24*time.Hour),
		now.Add(27*24*time.Hour))

	exec(ctx, tx,
		`INSERT INTO standing_approvals (standing_approval_id, agent_id, user_id, action_type, status, constraints, starts_at, expires_at, expired_at)
		 VALUES ($1, $2, $3, $4, 'expired', $5, $6, $7, $7)`,
		"sa-everything-expired", github, userHasEverything,
		"github.merge_pr",
		`{"pr": "supersuit-tech/permission-slip#123"}`,
		now.Add(-60*24*time.Hour),
		now.Add(-30*24*time.Hour))

	// ---------------------------------------------------------------
	// Action configurations (4: mix of active/disabled, with/without credentials)
	// ---------------------------------------------------------------

	// Claude Code → github.create_issue (locked repo, wildcard title/body)
	exec(ctx, tx,
		`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, status, name, description)
		 VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8)`,
		"ac-claude-create-issue", claude, userHasEverything,
		connectorGitHub, "github.create_issue",
		`{"repo": "supersuit-tech/permission-slip-web", "title": "*", "body": "*", "label": "bug"}`,
		"Create bug issues in permission-slip-web",
		"Agent can create issues labeled 'bug' in the main repo. Title and body are freeform.")

	// Claude Code → github.merge_pr (locked repo, wildcard PR number)
	exec(ctx, tx,
		`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, status, name, description)
		 VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8)`,
		"ac-claude-merge-pr", claude, userHasEverything,
		connectorGitHub, "github.merge_pr",
		`{"repo": "supersuit-tech/permission-slip-web", "pr": "*"}`,
		"Merge PRs in permission-slip-web",
		"Agent can merge any PR in the main repo.")

	// Slack Notifier → slack.send_message (locked channel, wildcard message)
	exec(ctx, tx,
		`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, status, name, description)
		 VALUES ($1, $2, $3, $4, $5, $6, 'active', $7, $8)`,
		"ac-slack-send-releases", slack, userHasEverything,
		connectorSlack, "slack.send_message",
		`{"channel": "#releases", "message": "*"}`,
		"Post to #releases",
		"Agent can send any message to the #releases channel.")

	// Disabled config — Datadog → slack.send_message (no credential yet)
	exec(ctx, tx,
		`INSERT INTO action_configurations (id, agent_id, user_id, connector_id, action_type, parameters, status, name, description)
		 VALUES ($1, $2, $3, $4, $5, $6, 'disabled', $7, $8)`,
		"ac-datadog-send-alerts", datadog, userHasEverything,
		connectorSlack, "slack.send_message",
		`{"channel": "#incidents", "message": "*"}`,
		"Post to #incidents (disabled)",
		"Disabled — credential needs to be assigned before activation.")

	// ---------------------------------------------------------------
	// Registration invite (active)
	// ---------------------------------------------------------------
	exec(ctx, tx,
		`INSERT INTO registration_invites (id, user_id, invite_code_hash, status, expires_at)
		 VALUES ($1, $2, $3, 'active', $4)`,
		"invite-everything-1", userHasEverything,
		"seedhash_invite-everything-1",
		now.Add(1*time.Hour))

	// Openclaw agent (if TEST_AGENT_PUB_KEY is set)
	if testAgentPubKey := os.Getenv("TEST_AGENT_PUB_KEY"); testAgentPubKey != "" {
		seedOpenclawAgent(ctx, tx, userHasEverything, testAgentPubKey)
	}

	if os.Getenv("TEST_AGENT_PUB_KEY") != "" {
		fmt.Println("  ✓ has-everything@test.local (11 agents (incl. Openclaw), 13 agent-connectors, 29 approvals, 2 credentials, 2 oauth connections, 7 agent-connector-credentials, 3 standing approvals, 4 action configs, 1 invite)")
	} else {
		fmt.Println("  ✓ has-everything@test.local (10 agents, 13 agent-connectors, 29 approvals, 2 credentials, 2 oauth connections, 7 agent-connector-credentials, 3 standing approvals, 4 action configs, 1 invite)")
	}
}

// ---------------------------------------------------------------------------
// 6. has-pending-approvals@test.local — focused on pending approvals
// ---------------------------------------------------------------------------

func seedUserHasPendingApprovals(ctx context.Context, tx db.DBTX, supa *supabaseClient) {
	insertUser(ctx, tx, supa, userHasPendingApproval, "has-pending-approvals@test.local")

	now := time.Now()

	// 2 registered agents to own the approvals
	agentClaude := insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, last_active_at, created_at)
		 VALUES ($1, $2, 'registered', $3, $4, $5, $4)
		 RETURNING agent_id`,
		"pk-seed-pending-claude", userHasPendingApproval,
		`{"name": "Claude Code"}`,
		now.Add(-14*24*time.Hour),
		now.Add(-10*time.Minute))

	agentGitHub := insertAgent(ctx, tx,
		`INSERT INTO agents (public_key, approver_id, status, metadata, registered_at, last_active_at, created_at)
		 VALUES ($1, $2, 'registered', $3, $4, $5, $4)
		 RETURNING agent_id`,
		"pk-seed-pending-github", userHasPendingApproval,
		`{"name": "GitHub Actions Bot"}`,
		now.Add(-7*24*time.Hour),
		now.Add(-30*time.Minute))

	// 4 pending approvals (24h expiry so they survive dev sessions)
	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-pa-pending-1", agentClaude, userHasPendingApproval,
		`{"type": "github.merge_pr", "parameters": {"repo": "supersuit-tech/webapp", "pr": 142}}`,
		`{"description": "Merge the authentication refactor PR"}`,
		now.Add(24*time.Hour),
		now.Add(-20*time.Minute))

	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-pa-pending-2", agentClaude, userHasPendingApproval,
		`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/api", "title": "Refactor user service"}}`,
		`{"description": "Create tracking issue for user service refactor"}`,
		now.Add(24*time.Hour),
		now.Add(-15*time.Minute))

	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-pa-pending-3", agentGitHub, userHasPendingApproval,
		`{"type": "slack.send_message", "parameters": {"channel": "#deployments", "message": "Deploying v3.0.0 to staging"}}`,
		`{"description": "Notify team about staging deployment"}`,
		now.Add(24*time.Hour),
		now.Add(-8*time.Minute))

	exec(ctx, tx,
		`INSERT INTO approvals (approval_id, agent_id, approver_id, action, context, status, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)`,
		"approval-pa-pending-4", agentGitHub, userHasPendingApproval,
		`{"type": "github.create_issue", "parameters": {"repo": "supersuit-tech/webapp", "title": "CI pipeline timeout on main"}}`,
		`{"description": "Auto-create issue from failed CI run #891"}`,
		now.Add(24*time.Hour),
		now.Add(-3*time.Minute))

	// Openclaw agent (if TEST_AGENT_PUB_KEY is set)
	if testAgentPubKey := os.Getenv("TEST_AGENT_PUB_KEY"); testAgentPubKey != "" {
		seedOpenclawAgent(ctx, tx, userHasPendingApproval, testAgentPubKey)
	}

	if os.Getenv("TEST_AGENT_PUB_KEY") != "" {
		fmt.Println("  ✓ has-pending-approvals@test.local (3 agents (incl. Openclaw), 4 pending approvals)")
	} else {
		fmt.Println("  ✓ has-pending-approvals@test.local (2 agents, 4 pending approvals)")
	}
}
