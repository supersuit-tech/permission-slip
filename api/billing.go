package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	pstripe "github.com/supersuit-tech/permission-slip-web/stripe"
)

// ── Response types ──────────────────────────────────────────────────────────

type subscriptionResponse struct {
	PlanID   string `json:"plan_id"`
	PlanName string `json:"plan_name"`
	billingSubscription
	PlanLimits planLimits `json:"plan_limits"`
	Usage      *usageInfo `json:"usage,omitempty"`
}

type planLimits struct {
	MaxRequestsPerMonth  *int `json:"max_requests_per_month"`
	MaxAgents            *int `json:"max_agents"`
	MaxStandingApprovals *int `json:"max_standing_approvals"`
	MaxCredentials       *int `json:"max_credentials"`
	AuditRetentionDays   int  `json:"audit_retention_days"`
}

type usageInfo struct {
	RequestCount int  `json:"request_count"`
	SMSCount     int  `json:"sms_count"`
	RequestLimit *int `json:"request_limit"`
	OverLimit    bool `json:"over_limit"`
}

type checkoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

// usageResponse is the JSON shape returned by GET /billing/usage.
// It provides detailed usage metrics for a billing period, including
// request/SMS totals, overage calculations, and an optional breakdown
// by agent, connector, and action type.
type usageResponse struct {
	PeriodStart time.Time          `json:"period_start"`
	PeriodEnd   time.Time          `json:"period_end"`
	Requests    requestsUsage      `json:"requests"`
	SMS         smsUsage           `json:"sms"`
	Breakdown   *usageBreakdownDTO `json:"breakdown,omitempty"`
}

// usageBreakdownDTO maps identifiers (agent ID, connector ID, action type)
// to their respective request counts within a billing period. Omitted from
// the response when no breakdown data has been recorded.
type usageBreakdownDTO struct {
	ByAgent      map[string]int `json:"by_agent,omitempty"`
	ByConnector  map[string]int `json:"by_connector,omitempty"`
	ByActionType map[string]int `json:"by_action_type,omitempty"`
}

// requestsUsage holds request count metrics for a billing period.
// CostCents is the estimated overage cost at $0.005/request, rounded up.
type requestsUsage struct {
	Total     int `json:"total"`
	Included  int `json:"included"`
	Overage   int `json:"overage"`
	CostCents int `json:"cost_cents"`
}

// smsUsage holds SMS metrics for a billing period.
// CostCents is estimated at the US/CA rate ($0.01/message).
type smsUsage struct {
	Total     int `json:"total"`
	CostCents int `json:"cost_cents"`
}

type invoiceListResponse struct {
	Invoices []apiInvoice `json:"invoices"`
	HasMore  bool         `json:"has_more"`
}

// apiInvoice is the API representation of an invoice, matching the OpenAPI Invoice schema.
type apiInvoice struct {
	ID               string  `json:"id"`
	Date             string  `json:"date"`
	AmountCents      int64   `json:"amount_cents"`
	Status           string  `json:"status"`
	PeriodStart      *string `json:"period_start,omitempty"`
	PeriodEnd        *string `json:"period_end,omitempty"`
	StripeInvoiceURL *string `json:"stripe_invoice_url,omitempty"`
}

// toAPIInvoice converts a Stripe InvoiceSummary to the API Invoice schema.
func toAPIInvoice(s pstripe.InvoiceSummary) apiInvoice {
	inv := apiInvoice{
		ID:          s.ID,
		Date:        time.Unix(s.Created, 0).UTC().Format(time.RFC3339),
		AmountCents: s.AmountPaid,
		Status:      s.Status,
	}
	if s.PeriodStart > 0 {
		ps := time.Unix(s.PeriodStart, 0).UTC().Format(time.RFC3339)
		inv.PeriodStart = &ps
	}
	if s.PeriodEnd > 0 {
		pe := time.Unix(s.PeriodEnd, 0).UTC().Format(time.RFC3339)
		inv.PeriodEnd = &pe
	}
	if s.HostedURL != "" {
		hu := s.HostedURL
		inv.StripeInvoiceURL = &hu
	}
	return inv
}

type billingPlanResponse struct {
	Plan         billingPlan         `json:"plan"`
	Subscription billingSubscription `json:"subscription"`
	Usage        billingUsageSummary `json:"usage"`
}

type billingPlan struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	planLimits
}

type billingSubscription struct {
	Status             string     `json:"status"`
	CurrentPeriodStart time.Time  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `json:"current_period_end"`
	HasPaymentMethod   bool       `json:"has_payment_method"`
	CanUpgrade         bool       `json:"can_upgrade"`
	CanDowngrade       bool       `json:"can_downgrade"`
	GracePeriodEndsAt  *time.Time `json:"grace_period_ends_at"`
}

type billingUsageSummary struct {
	Requests          int `json:"requests"`
	Agents            int `json:"agents"`
	StandingApprovals int `json:"standing_approvals"`
	Credentials       int `json:"credentials"`
}

// newBillingSubscription builds subscription status fields from a DB subscription.
func newBillingSubscription(sub *db.SubscriptionWithPlan) billingSubscription {
	return billingSubscription{
		Status:             string(sub.Status),
		CurrentPeriodStart: sub.CurrentPeriodStart,
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		HasPaymentMethod:   sub.StripeCustomerID != nil,
		CanUpgrade:         sub.PlanID == db.PlanFree,
		CanDowngrade:       sub.PlanID == db.PlanPayAsYouGo,
		GracePeriodEndsAt:  sub.GracePeriodEndsAt(),
	}
}

// newPlanLimits builds plan limit fields from a DB plan.
func newPlanLimits(p *db.Plan) planLimits {
	return planLimits{
		MaxRequestsPerMonth:  p.MaxRequestsPerMonth,
		MaxAgents:            p.MaxAgents,
		MaxStandingApprovals: p.MaxStandingApprovals,
		MaxCredentials:       p.MaxCredentials,
		AuditRetentionDays:   p.AuditRetentionDays,
	}
}

type downgradeResponse struct {
	Status            string     `json:"status"`
	PlanID            string     `json:"plan_id"`
	DowngradedAt      *time.Time `json:"downgraded_at"`
	GracePeriodEndsAt *time.Time `json:"grace_period_ends_at"`
}

// gracePeriodEnd returns the time when the 7-day grace period expires for a
// given downgrade timestamp, or nil if no downgrade timestamp is set.
func gracePeriodEnd(downgradedAt *time.Time) *time.Time {
	if downgradedAt == nil {
		return nil
	}
	t := downgradedAt.Add(db.DowngradeGracePeriod)
	return &t
}

// maxInvoiceResults is the maximum number of invoices returned by the list endpoint.
const maxInvoiceResults = 24

// ── Route registration ──────────────────────────────────────────────────────

// RegisterBillingRoutes adds billing endpoints to the mux.
// These are only registered when billing is enabled.
func init() {
	RegisterRouteGroup(RegisterBillingRoutes)
}

func RegisterBillingRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /billing/plan", requireProfile(handleGetBillingPlan(deps)))
	mux.Handle("GET /billing/subscription", requireProfile(handleGetSubscription(deps)))
	mux.Handle("GET /billing/usage", requireProfile(handleGetUsage(deps)))
	// Deprecated: use POST /billing/upgrade instead. Kept for backward compatibility.
	mux.Handle("POST /billing/checkout", requireProfile(handleCreateCheckout(deps)))
	mux.Handle("POST /billing/upgrade", requireProfile(handleCreateCheckout(deps)))
	mux.Handle("POST /billing/downgrade", requireProfile(handleDowngrade(deps)))
	mux.Handle("GET /billing/invoices", requireProfile(handleListInvoices(deps)))
	mux.Handle("POST /billing/activate", requireProfile(handleActivateUpgrade(deps)))
}

// ── GET /billing/plan ────────────────────────────────────────────────────────

func handleGetBillingPlan(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		sub, err := db.GetSubscriptionWithPlan(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] GetBillingPlan: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}
		if sub == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrSubscriptionNotFound, "No subscription found"))
			return
		}

		resp := billingPlanResponse{
			Plan: billingPlan{
				ID:         sub.PlanID,
				Name:       sub.Plan.Name,
				planLimits: newPlanLimits(&sub.Plan),
			},
			Subscription: newBillingSubscription(sub),
		}

		// Gather usage counts (non-fatal — return zeros on error).
		usage, err := db.GetCurrentPeriodUsage(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] GetBillingPlan: usage lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		}
		if usage != nil {
			resp.Usage.Requests = usage.RequestCount
		}

		if count, err := db.CountRegisteredAgentsByUser(r.Context(), deps.DB, profile.ID); err != nil {
			log.Printf("[%s] GetBillingPlan: count agents: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		} else {
			resp.Usage.Agents = count
		}

		if count, err := db.CountActiveStandingApprovalsByUser(r.Context(), deps.DB, profile.ID); err != nil {
			log.Printf("[%s] GetBillingPlan: count standing approvals: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		} else {
			resp.Usage.StandingApprovals = count
		}

		if count, err := db.CountCredentialsByUser(r.Context(), deps.DB, profile.ID); err != nil {
			log.Printf("[%s] GetBillingPlan: count credentials: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		} else {
			resp.Usage.Credentials = count
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// ── GET /billing/subscription ───────────────────────────────────────────────

func handleGetSubscription(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		sub, err := db.GetSubscriptionWithPlan(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] GetSubscription: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}
		if sub == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrSubscriptionNotFound, "No subscription found"))
			return
		}

		resp := subscriptionResponse{
			PlanID:              sub.PlanID,
			PlanName:            sub.Plan.Name,
			billingSubscription: newBillingSubscription(sub),
			PlanLimits:          newPlanLimits(&sub.Plan),
		}

		// Attach current period usage if available (non-fatal — subscription
		// data is still returned without usage on error).
		usage, err := db.GetCurrentPeriodUsage(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] GetSubscription: usage lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		}
		if usage != nil {
			ui := &usageInfo{
				RequestCount: usage.RequestCount,
				SMSCount:     usage.SMSCount,
				RequestLimit: sub.Plan.MaxRequestsPerMonth,
			}
			if sub.Plan.MaxRequestsPerMonth != nil && usage.RequestCount > *sub.Plan.MaxRequestsPerMonth {
				ui.OverLimit = true
			}
			resp.Usage = ui
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// ── POST /billing/checkout ──────────────────────────────────────────────────

func handleCreateCheckout(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		// Check business logic before infrastructure dependencies so users
		// get accurate error messages (e.g. "already subscribed" not "billing
		// not configured").
		sub, err := db.GetSubscriptionByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] CreateCheckout: subscription lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}

		// If already on paid plan, reject.
		if sub != nil && sub.PlanID == db.PlanPayAsYouGo {
			RespondError(w, r, http.StatusConflict, Conflict(ErrAlreadySubscribed, "Already subscribed to a paid plan"))
			return
		}

		if deps.Stripe == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Billing not configured"))
			return
		}

		var stripeCustomerID string

		if sub != nil && sub.StripeCustomerID != nil {
			// Reuse existing Stripe customer.
			stripeCustomerID = *sub.StripeCustomerID
		} else {
			// Create a new Stripe customer.
			email := ""
			if profile.Email != nil {
				email = *profile.Email
			}

			cust, err := deps.Stripe.CreateCustomer(r.Context(), email, profile.ID)
			if err != nil {
				log.Printf("[%s] CreateCheckout: create customer: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to create Stripe customer"))
				return
			}
			stripeCustomerID = cust.ID

			// Persist Stripe customer ID so subsequent checkout attempts reuse
			// the same customer instead of creating duplicates.
			if _, err := db.UpdateSubscriptionStripe(r.Context(), deps.DB, profile.ID, &stripeCustomerID, nil); err != nil {
				log.Printf("[%s] CreateCheckout: save customer ID: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
			}
		}

		// Build success/cancel URLs. Include {CHECKOUT_SESSION_ID} so the
		// frontend can call POST /billing/activate to confirm the upgrade
		// without relying solely on the webhook.
		successURL := deps.BaseURL + "/settings/billing?upgraded=true&session_id={CHECKOUT_SESSION_ID}"
		cancelURL := deps.BaseURL + "/settings/billing"

		sess, err := deps.Stripe.CreateCheckoutSession(r.Context(), stripeCustomerID, successURL, cancelURL)
		if err != nil {
			log.Printf("[%s] CreateCheckout: create session: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to create checkout session"))
			return
		}

		RespondJSON(w, http.StatusOK, checkoutResponse{CheckoutURL: sess.URL})
	}
}

// ── GET /billing/usage ─────────────────────────────────────────────────────

func handleGetUsage(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		sub, err := db.GetSubscriptionWithPlan(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] GetUsage: subscription lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}
		if sub == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrSubscriptionNotFound, "No subscription found"))
			return
		}

		// Determine the target billing period. If period_start is provided,
		// look up that specific period; otherwise default to the current one.
		var periodStart time.Time
		if ps := r.URL.Query().Get("period_start"); ps != "" {
			parsed, parseErr := time.Parse(time.RFC3339, ps)
			if parseErr != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "period_start must be a valid ISO 8601 date-time"))
				return
			}
			periodStart = parsed
		}

		resp := usageResponse{
			PeriodStart: sub.CurrentPeriodStart,
			PeriodEnd:   sub.CurrentPeriodEnd,
		}

		// Determine included request allowance.
		// Paid plans get the same free allowance as the free tier (250 requests).
		included := int(pstripe.FreeRequestAllowance())
		if sub.Plan.MaxRequestsPerMonth != nil {
			included = *sub.Plan.MaxRequestsPerMonth
		}
		resp.Requests.Included = included

		var usage *db.UsagePeriod
		if periodStart.IsZero() {
			// Current billing period.
			usage, err = db.GetCurrentPeriodUsage(r.Context(), deps.DB, profile.ID)
		} else {
			// Historical period — update response bounds to match the query.
			_, periodEnd := db.BillingPeriodBounds(periodStart)
			resp.PeriodStart = periodStart
			resp.PeriodEnd = periodEnd
			usage, err = db.GetUsageByPeriod(r.Context(), deps.DB, profile.ID, periodStart)
		}
		if err != nil {
			log.Printf("[%s] GetUsage: usage lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch usage data"))
			return
		}
		if usage != nil {
			resp.Requests.Total = usage.RequestCount
			resp.SMS.Total = usage.SMSCount

			// Calculate overage using the shared pricing function.
			if included > 0 && usage.RequestCount > included {
				resp.Requests.Overage = usage.RequestCount - included
				resp.Requests.CostCents = pstripe.OverageCostCents(resp.Requests.Overage)
			}

			// SMS cost: $0.01/message (us_ca rate).
			resp.SMS.CostCents = usage.SMSCount

			// Include breakdown if available.
			b := usage.ParseBreakdown()
			if len(b.ByAgent) > 0 || len(b.ByConnector) > 0 || len(b.ByActionType) > 0 {
				resp.Breakdown = &usageBreakdownDTO{
					ByAgent:      b.ByAgent,
					ByConnector:  b.ByConnector,
					ByActionType: b.ByActionType,
				}
			}
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// ── POST /billing/downgrade ───────────────────────────────────────────────

func handleDowngrade(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		sub, err := db.GetSubscriptionByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] Downgrade: subscription lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}
		if sub == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrSubscriptionNotFound, "No subscription found"))
			return
		}

		// Can only downgrade from a paid plan.
		if sub.PlanID == db.PlanFree {
			RespondError(w, r, http.StatusConflict, Conflict(ErrAlreadyDowngraded, "Already on the free plan"))
			return
		}

		// Check plan limits: count agents, standing approvals, credentials.
		freePlan := db.GetPlan(db.PlanFree)
		if freePlan == nil {
			log.Printf("[%s] Downgrade: free plan not found in config", TraceID(r.Context()))
			RespondError(w, r, http.StatusInternalServerError, InternalError("Free plan not configured"))
			return
		}

		if limitErr := validateDowngradeLimits(r.Context(), deps, profile.ID, freePlan); limitErr != nil {
			status := http.StatusConflict
			if limitErr.Error.Code == ErrInternalError {
				status = http.StatusInternalServerError
			}
			RespondError(w, r, status, *limitErr)
			return
		}

		// Cancel Stripe subscription if one exists.
		if deps.Stripe != nil && sub.StripeSubscriptionID != nil {
			if _, err := deps.Stripe.CancelSubscription(r.Context(), *sub.StripeSubscriptionID); err != nil {
				log.Printf("[%s] Downgrade: cancel Stripe subscription: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to cancel subscription with payment provider"))
				return
			}
		}

		// Downgrade plan locally.
		updated, err := db.UpdateSubscriptionPlan(r.Context(), deps.DB, profile.ID, db.PlanFree)
		if err != nil {
			log.Printf("[%s] Downgrade: update plan: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to downgrade plan"))
			return
		}
		if updated == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrSubscriptionNotFound, "No subscription found"))
			return
		}

		RespondJSON(w, http.StatusOK, downgradeResponse{
			Status:            string(updated.Status),
			PlanID:            updated.PlanID,
			DowngradedAt:      updated.DowngradedAt,
			GracePeriodEndsAt: gracePeriodEnd(updated.DowngradedAt),
		})
	}
}

// validateDowngradeLimits checks that the user's current resource counts are
// within the free plan's limits. Returns an ErrorResponse if any limit is
// exceeded or if a count cannot be verified (fail-closed), nil otherwise.
func validateDowngradeLimits(ctx context.Context, deps *Deps, userID string, freePlan *db.Plan) *ErrorResponse {
	if freePlan.MaxAgents != nil {
		count, err := db.CountRegisteredAgentsByUser(ctx, deps.DB, userID)
		if err != nil {
			log.Printf("[%s] Downgrade: count agents: %v", TraceID(ctx), err)
			CaptureError(ctx, err)
			resp := InternalError("Unable to verify agent count")
			return &resp
		}
		if count > *freePlan.MaxAgents {
			resp := Conflict(ErrDowngradeLimitExceeded, "Too many active agents for the free plan")
			resp.Error.Details = map[string]any{
				"resource":    "agents",
				"current":     count,
				"max_allowed": *freePlan.MaxAgents,
			}
			return &resp
		}
	}

	if freePlan.MaxStandingApprovals != nil {
		count, err := db.CountActiveStandingApprovalsByUser(ctx, deps.DB, userID)
		if err != nil {
			log.Printf("[%s] Downgrade: count standing approvals: %v", TraceID(ctx), err)
			CaptureError(ctx, err)
			resp := InternalError("Unable to verify standing approval count")
			return &resp
		}
		if count > *freePlan.MaxStandingApprovals {
			resp := Conflict(ErrDowngradeLimitExceeded, "Too many active standing approvals for the free plan")
			resp.Error.Details = map[string]any{
				"resource":    "standing_approvals",
				"current":     count,
				"max_allowed": *freePlan.MaxStandingApprovals,
			}
			return &resp
		}
	}

	if freePlan.MaxCredentials != nil {
		count, err := db.CountCredentialsByUser(ctx, deps.DB, userID)
		if err != nil {
			log.Printf("[%s] Downgrade: count credentials: %v", TraceID(ctx), err)
			CaptureError(ctx, err)
			resp := InternalError("Unable to verify credential count")
			return &resp
		}
		if count > *freePlan.MaxCredentials {
			resp := Conflict(ErrDowngradeLimitExceeded, "Too many stored credentials for the free plan")
			resp.Error.Details = map[string]any{
				"resource":    "credentials",
				"current":     count,
				"max_allowed": *freePlan.MaxCredentials,
			}
			return &resp
		}
	}

	return nil
}

// ── GET /billing/invoices ─────────────────────────────────────────────────

func handleListInvoices(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		if deps.Stripe == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Billing not configured"))
			return
		}

		sub, err := db.GetSubscriptionByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ListInvoices: subscription lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}
		if sub == nil || sub.StripeCustomerID == nil {
			// No Stripe customer → no invoices.
			RespondJSON(w, http.StatusOK, invoiceListResponse{Invoices: []apiInvoice{}})
			return
		}

		summaries, hasMore, err := deps.Stripe.ListInvoices(r.Context(), *sub.StripeCustomerID, maxInvoiceResults)
		if err != nil {
			log.Printf("[%s] ListInvoices: Stripe list: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to fetch invoices"))
			return
		}

		invoices := make([]apiInvoice, 0, len(summaries))
		for _, s := range summaries {
			invoices = append(invoices, toAPIInvoice(s))
		}

		RespondJSON(w, http.StatusOK, invoiceListResponse{Invoices: invoices, HasMore: hasMore})
	}
}

// upstreamError returns a 502 ErrorResponse for upstream Stripe failures.
func upstreamError(message string) ErrorResponse {
	return newErrorResponse(ErrUpstreamError, message, true)
}

// ReportPeriodUsage creates Stripe Invoice Items for a user's billing period
// usage (requests + SMS). Called by the webhook handler when a billing period
// ends or by a background cron.
func ReportPeriodUsage(ctx context.Context, deps *Deps, userID string, usage *db.UsagePeriod) {
	if deps.Stripe == nil {
		return
	}

	sub, err := db.GetSubscriptionByUserID(ctx, deps.DB, userID)
	if err != nil {
		log.Printf("billing: ReportPeriodUsage: subscription lookup for %s: %v", userID, err)
		return
	}
	if sub == nil || sub.StripeCustomerID == nil {
		return
	}

	stripeCustomerID := *sub.StripeCustomerID

	// Report request usage.
	if usage.RequestCount > 0 {
		if _, err := deps.Stripe.CreateUsageInvoiceItem(ctx, stripeCustomerID, int64(usage.RequestCount)); err != nil {
			log.Printf("billing: report request usage for %s: %v", userID, err)
		}
	}

	// Report SMS usage. For now, we bill all SMS at the base "us_ca" rate.
	// A future iteration can track per-destination counts in the breakdown.
	if usage.SMSCount > 0 {
		if _, err := deps.Stripe.CreateSMSInvoiceItem(ctx, stripeCustomerID, "us_ca", int64(usage.SMSCount)); err != nil {
			log.Printf("billing: report SMS usage for %s: %v", userID, err)
		}
	}
}
