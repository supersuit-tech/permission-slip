package api

import (
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ── Response types ──────────────────────────────────────────────────────────

type paymentMethodResponse struct {
	ID                  string     `json:"id"`
	Label               string     `json:"label"`
	Brand               string     `json:"brand"`
	Last4               string     `json:"last4"`
	ExpMonth            int        `json:"exp_month"`
	ExpYear             int        `json:"exp_year"`
	IsDefault           bool       `json:"is_default"`
	PerTransactionLimit *int       `json:"per_transaction_limit"`
	MonthlyLimit        *int       `json:"monthly_limit"`
	MonthlySpend        *int       `json:"monthly_spend,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type paymentMethodListResponse struct {
	PaymentMethods []paymentMethodResponse `json:"payment_methods"`
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
	ClearPerTxLimit     *bool   `json:"clear_per_transaction_limit,omitempty"`
	ClearMonthlyLimit   *bool   `json:"clear_monthly_limit,omitempty"`
}

type deletePaymentMethodResponse struct {
	Deleted bool `json:"deleted"`
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

		resp := paymentMethodListResponse{
			PaymentMethods: make([]paymentMethodResponse, 0, len(methods)),
		}
		for i := range methods {
			pmr := toPaymentMethodResponse(&methods[i])
			// Include monthly spend if the method has a monthly limit.
			if methods[i].MonthlyLimit != nil {
				spend, err := db.GetMonthlySpend(r.Context(), deps.DB, methods[i].ID)
				if err != nil {
					log.Printf("[%s] ListPaymentMethods: monthly spend lookup: %v", TraceID(r.Context()), err)
					CaptureError(r.Context(), err)
				} else {
					pmr.MonthlySpend = &spend
				}
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

			// Persist the Stripe customer ID.
			if sub != nil {
				if _, err := db.UpdateSubscriptionStripe(r.Context(), deps.DB, profile.ID, &stripeCustomerID, nil); err != nil {
					log.Printf("[%s] CreateSetupIntent: save customer ID: %v", TraceID(r.Context()), err)
					CaptureError(r.Context(), err)
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

		// If setting as default, clear existing default first.
		if req.IsDefault {
			if err := db.ClearDefaultPaymentMethod(r.Context(), deps.DB, profile.ID); err != nil {
				log.Printf("[%s] ConfirmPaymentMethod: clear default: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
			}
		}

		created, err := db.CreatePaymentMethod(r.Context(), deps.DB, pm)
		if err != nil {
			log.Printf("[%s] ConfirmPaymentMethod: create: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save payment method"))
			return
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

		// If setting as default, clear existing default first.
		if req.IsDefault != nil && *req.IsDefault {
			if err := db.ClearDefaultPaymentMethod(r.Context(), deps.DB, profile.ID); err != nil {
				log.Printf("[%s] UpdatePaymentMethod: clear default: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
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
		if req.ClearPerTxLimit != nil && *req.ClearPerTxLimit {
			params.ClearPerTxLimit = true
		}
		if req.ClearMonthlyLimit != nil && *req.ClearMonthlyLimit {
			params.ClearMonthlyLimit = true
		}

		updated, err := db.UpdatePaymentMethod(r.Context(), deps.DB, profile.ID, pmID, params)
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

		RespondJSON(w, http.StatusOK, deletePaymentMethodResponse{Deleted: true})
	}
}
