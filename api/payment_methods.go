package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// maxLabelLength is the maximum length of a payment method label.
const maxLabelLength = 100

// ── Response types ──────────────────────────────────────────────────────────

type paymentMethodResponse struct {
	ID                  string    `json:"id"`
	Label               string    `json:"label"`
	Brand               string    `json:"brand"`
	Last4               string    `json:"last4"`
	ExpMonth            int       `json:"exp_month"`
	ExpYear             int       `json:"exp_year"`
	IsDefault           bool      `json:"is_default"`
	PerTransactionLimit *int      `json:"per_transaction_limit"`
	MonthlyLimit        *int      `json:"monthly_limit"`
	MonthlySpend        *int      `json:"monthly_spend,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type paymentMethodListResponse struct {
	PaymentMethods []paymentMethodResponse `json:"payment_methods"`
	MaxAllowed     int                     `json:"max_allowed"`
}

type setupIntentResponse struct {
	ClientSecret string `json:"client_secret"`
}

type confirmPaymentMethodRequest struct {
	PaymentMethodID string `json:"payment_method_id"` // Stripe pm_* ID
	Label           string `json:"label,omitempty"`
	IsDefault       bool   `json:"is_default"`
}

type updatePaymentMethodRequest struct {
	Label               *string `json:"label,omitempty"`
	IsDefault           *bool   `json:"is_default,omitempty"`
	PerTransactionLimit *int    `json:"per_transaction_limit,omitempty"`
	MonthlyLimit        *int    `json:"monthly_limit,omitempty"`
	ClearLabel          *bool   `json:"clear_label,omitempty"`
	ClearPerTxLimit     *bool   `json:"clear_per_transaction_limit,omitempty"`
	ClearMonthlyLimit   *bool   `json:"clear_monthly_limit,omitempty"`
}

type deletePaymentMethodResponse struct {
	Deleted        bool `json:"deleted"`
	AffectedAgents int  `json:"affected_agents"`
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func toPaymentMethodResponse(pm *db.PaymentMethod) paymentMethodResponse {
	return paymentMethodResponse{
		ID:                  pm.ID,
		Label:               pm.Label,
		Brand:               pm.Brand,
		Last4:               pm.Last4,
		ExpMonth:            pm.ExpMonth,
		ExpYear:             pm.ExpYear,
		IsDefault:           pm.IsDefault,
		PerTransactionLimit: pm.PerTransactionLimit,
		MonthlyLimit:        pm.MonthlyLimit,
		CreatedAt:           pm.CreatedAt,
		UpdatedAt:           pm.UpdatedAt,
	}
}

// ── Route registration ──────────────────────────────────────────────────────

func init() {
	RegisterRouteGroup(RegisterPaymentMethodRoutes)
}

// RegisterPaymentMethodRoutes adds payment method endpoints to the mux.
func RegisterPaymentMethodRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /payment-methods", requireProfile(handleListPaymentMethods(deps)))
	mux.Handle("POST /payment-methods/setup-intent", requireProfile(handleCreateSetupIntent(deps)))
	mux.Handle("POST /payment-methods", requireProfile(handleConfirmPaymentMethod(deps)))
	mux.Handle("PATCH /payment-methods/{id}", requireProfile(handleUpdatePaymentMethod(deps)))
	mux.Handle("DELETE /payment-methods/{id}", requireProfile(handleDeletePaymentMethod(deps)))
}

// ── GET /payment-methods ────────────────────────────────────────────────────

func handleListPaymentMethods(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		methods, err := db.ListPaymentMethodsByUser(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ListPaymentMethods: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list payment methods"))
			return
		}

		// Batch-fetch monthly spend for all payment methods in a single query.
		allIDs := make([]string, len(methods))
		for i := range methods {
			allIDs[i] = methods[i].ID
		}
		spendMap, err := db.GetMonthlySpendBatch(r.Context(), deps.DB, allIDs)
		if err != nil {
			log.Printf("[%s] ListPaymentMethods: monthly spend batch lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			// Non-fatal — continue without spend data.
		}

		resp := paymentMethodListResponse{
			PaymentMethods: make([]paymentMethodResponse, 0, len(methods)),
			MaxAllowed:     maxPaymentMethodsPerUser,
		}
		for i := range methods {
			pmr := toPaymentMethodResponse(&methods[i])
			// Always include monthly spend so the frontend can show usage.
			if spendMap != nil {
				spend := spendMap[methods[i].ID] // defaults to 0 if not in map
				pmr.MonthlySpend = &spend
			}
			resp.PaymentMethods = append(resp.PaymentMethods, pmr)
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// ── POST /payment-methods/setup-intent ──────────────────────────────────────

func handleCreateSetupIntent(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		if deps.Stripe == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Payment processing not configured"))
			return
		}

		// Get or create a Stripe customer for this user.
		sub, err := db.GetSubscriptionByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] CreateSetupIntent: subscription lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}

		var stripeCustomerID string
		if sub != nil && sub.StripeCustomerID != nil {
			stripeCustomerID = *sub.StripeCustomerID
		} else {
			// Create a new Stripe customer.
			email := ""
			if profile.Email != nil {
				email = *profile.Email
			}
			cust, err := deps.Stripe.CreateCustomer(r.Context(), email, profile.ID)
			if err != nil {
				log.Printf("[%s] CreateSetupIntent: create customer: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to create payment customer"))
				return
			}
			stripeCustomerID = cust.ID

			// Persist the Stripe customer ID on the subscription record.
			if sub != nil {
				if _, err := db.UpdateSubscriptionStripe(r.Context(), deps.DB, profile.ID, &stripeCustomerID, nil); err != nil {
					log.Printf("[%s] CreateSetupIntent: save customer ID: %v", TraceID(r.Context()), err)
					CaptureError(r.Context(), err)
				}
			} else {
				// No subscription exists yet — create a free-tier subscription
				// so the Stripe customer ID is persisted for future use.
				newSub, createErr := db.CreateSubscription(r.Context(), deps.DB, profile.ID, "free")
				if createErr != nil {
					log.Printf("[%s] CreateSetupIntent: create subscription: %v", TraceID(r.Context()), createErr)
					CaptureError(r.Context(), createErr)
				} else if newSub != nil {
					if _, updateErr := db.UpdateSubscriptionStripe(r.Context(), deps.DB, profile.ID, &stripeCustomerID, nil); updateErr != nil {
						log.Printf("[%s] CreateSetupIntent: save customer ID on new sub: %v", TraceID(r.Context()), updateErr)
						CaptureError(r.Context(), updateErr)
					}
				}
			}
		}

		si, err := deps.Stripe.CreateSetupIntent(r.Context(), stripeCustomerID)
		if err != nil {
			log.Printf("[%s] CreateSetupIntent: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to create setup intent"))
			return
		}

		RespondJSON(w, http.StatusOK, setupIntentResponse{ClientSecret: si.ClientSecret})
	}
}

// ── POST /payment-methods ───────────────────────────────────────────────────

// maxPaymentMethodsPerUser is the maximum number of stored payment methods
// a single user can have. Prevents abuse.
const maxPaymentMethodsPerUser = 10

func handleConfirmPaymentMethod(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req confirmPaymentMethodRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		if req.PaymentMethodID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "payment_method_id is required"))
			return
		}
		if !strings.HasPrefix(req.PaymentMethodID, "pm_") {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "payment_method_id must be a valid Stripe payment method ID"))
			return
		}
		if len(req.Label) > maxLabelLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "label exceeds maximum length"))
			return
		}

		if deps.Stripe == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Payment processing not configured"))
			return
		}

		// Check limit.
		count, err := db.CountPaymentMethodsByUser(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ConfirmPaymentMethod: count: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to check payment method count"))
			return
		}
		if count >= maxPaymentMethodsPerUser {
			RespondError(w, r, http.StatusConflict, Conflict(ErrConstraintViolation, "Maximum payment methods limit reached"))
			return
		}

		// Retrieve the payment method from Stripe to get card details.
		stripePM, err := deps.Stripe.GetPaymentMethod(r.Context(), req.PaymentMethodID)
		if err != nil {
			log.Printf("[%s] ConfirmPaymentMethod: get stripe PM: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to verify payment method"))
			return
		}

		// Build payment method record from Stripe data.
		pm := &db.PaymentMethod{
			UserID:                profile.ID,
			StripePaymentMethodID: stripePM.ID,
			Label:                 req.Label,
			IsDefault:             req.IsDefault,
		}

		if stripePM.Card != nil {
			pm.Brand = string(stripePM.Card.Brand)
			pm.Last4 = stripePM.Card.Last4
			pm.ExpMonth = int(stripePM.Card.ExpMonth)
			pm.ExpYear = int(stripePM.Card.ExpYear)
		}

		// Use a transaction to atomically clear old default + create new card.
		tx, owned, txErr := db.BeginOrContinue(r.Context(), deps.DB)
		if txErr != nil {
			log.Printf("[%s] ConfirmPaymentMethod: begin tx: %v", TraceID(r.Context()), txErr)
			CaptureError(r.Context(), txErr)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save payment method"))
			return
		}
		if owned {
			defer func() { _ = db.RollbackTx(r.Context(), tx) }()
		}

		if req.IsDefault {
			if err := db.ClearDefaultPaymentMethod(r.Context(), tx, profile.ID); err != nil {
				log.Printf("[%s] ConfirmPaymentMethod: clear default: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save payment method"))
				return
			}
		}

		created, err := db.CreatePaymentMethod(r.Context(), tx, pm)
		if err != nil {
			log.Printf("[%s] ConfirmPaymentMethod: create: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save payment method"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] ConfirmPaymentMethod: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save payment method"))
				return
			}
		}

		RespondJSON(w, http.StatusCreated, toPaymentMethodResponse(created))
	}
}

// ── PATCH /payment-methods/{id} ─────────────────────────────────────────────

func handleUpdatePaymentMethod(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		pmID := r.PathValue("id")
		if pmID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Payment method ID is required"))
			return
		}

		var req updatePaymentMethodRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		// Validate label length.
		if req.Label != nil && len(*req.Label) > maxLabelLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "label exceeds maximum length"))
			return
		}

		// Validate spending limits.
		if req.PerTransactionLimit != nil && *req.PerTransactionLimit < 0 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Per-transaction limit cannot be negative"))
			return
		}
		if req.MonthlyLimit != nil && *req.MonthlyLimit < 0 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Monthly limit cannot be negative"))
			return
		}
		if req.PerTransactionLimit != nil && req.MonthlyLimit != nil &&
			*req.PerTransactionLimit > *req.MonthlyLimit {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Per-transaction limit cannot exceed monthly limit"))
			return
		}

		// Use a transaction to atomically clear old default + update card.
		tx, owned, txErr := db.BeginOrContinue(r.Context(), deps.DB)
		if txErr != nil {
			log.Printf("[%s] UpdatePaymentMethod: begin tx: %v", TraceID(r.Context()), txErr)
			CaptureError(r.Context(), txErr)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update payment method"))
			return
		}
		if owned {
			defer func() { _ = db.RollbackTx(r.Context(), tx) }()
		}

		if req.IsDefault != nil && *req.IsDefault {
			if err := db.ClearDefaultPaymentMethod(r.Context(), tx, profile.ID); err != nil {
				log.Printf("[%s] UpdatePaymentMethod: clear default: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update payment method"))
				return
			}
		}

		params := db.UpdatePaymentMethodParams{
			Label:     req.Label,
			IsDefault: req.IsDefault,
		}
		if req.PerTransactionLimit != nil {
			params.PerTransactionLimit = req.PerTransactionLimit
		}
		if req.MonthlyLimit != nil {
			params.MonthlyLimit = req.MonthlyLimit
		}
		if req.ClearLabel != nil && *req.ClearLabel {
			params.ClearLabel = true
		}
		if req.ClearPerTxLimit != nil && *req.ClearPerTxLimit {
			params.ClearPerTxLimit = true
		}
		if req.ClearMonthlyLimit != nil && *req.ClearMonthlyLimit {
			params.ClearMonthlyLimit = true
		}

		updated, err := db.UpdatePaymentMethod(r.Context(), tx, profile.ID, pmID, params)
		if err != nil {
			log.Printf("[%s] UpdatePaymentMethod: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update payment method"))
			return
		}
		if updated == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrPaymentMethodNotFound, "Payment method not found"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] UpdatePaymentMethod: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update payment method"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, toPaymentMethodResponse(updated))
	}
}

// ── DELETE /payment-methods/{id} ────────────────────────────────────────────

func handleDeletePaymentMethod(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		pmID := r.PathValue("id")
		if pmID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Payment method ID is required"))
			return
		}

		// Look up the payment method to get the Stripe ID for detachment.
		pm, err := db.GetPaymentMethodByID(r.Context(), deps.DB, profile.ID, pmID)
		if err != nil {
			log.Printf("[%s] DeletePaymentMethod: lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to look up payment method"))
			return
		}
		if pm == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrPaymentMethodNotFound, "Payment method not found"))
			return
		}

		// Count affected agents before deletion (cascade will remove bindings).
		affectedAgents, err := db.CountAgentsByPaymentMethod(r.Context(), deps.DB, pmID)
		if err != nil {
			log.Printf("[%s] DeletePaymentMethod: count agents: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			// Non-fatal — continue with deletion.
			affectedAgents = 0
		}

		// Detach from Stripe customer (best-effort — we still delete locally even if this fails).
		if deps.Stripe != nil {
			if _, err := deps.Stripe.DetachPaymentMethod(r.Context(), pm.StripePaymentMethodID); err != nil {
				log.Printf("[%s] DeletePaymentMethod: Stripe detach: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
			}
		}

		deleted, err := db.DeletePaymentMethod(r.Context(), deps.DB, profile.ID, pmID)
		if err != nil {
			log.Printf("[%s] DeletePaymentMethod: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete payment method"))
			return
		}
		if !deleted {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrPaymentMethodNotFound, "Payment method not found"))
			return
		}

		RespondJSON(w, http.StatusOK, deletePaymentMethodResponse{Deleted: true, AffectedAgents: affectedAgents})
	}
}
