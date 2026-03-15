package api

import (
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// --- Response types ---

type agentPaymentMethodResponse struct {
	AgentID         int64     `json:"agent_id"`
	PaymentMethodID string    `json:"payment_method_id"`
	CreatedAt       time.Time `json:"created_at"`
}

type assignAgentPaymentMethodRequest struct {
	PaymentMethodID string `json:"payment_method_id"`
}

func init() {
	RegisterRouteGroup(RegisterAgentPaymentMethodRoutes)
}

// RegisterAgentPaymentMethodRoutes adds agent payment method endpoints to the mux.
func RegisterAgentPaymentMethodRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /agents/{agent_id}/payment-method", requireProfile(handleGetAgentPaymentMethod(deps)))
	mux.Handle("PUT /agents/{agent_id}/payment-method", requireProfile(handleAssignAgentPaymentMethod(deps)))
	mux.Handle("DELETE /agents/{agent_id}/payment-method", requireProfile(handleRemoveAgentPaymentMethod(deps)))
}

// ── GET /agents/{agent_id}/payment-method ───────────────────────────────────

func handleGetAgentPaymentMethod(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		binding, err := db.GetAgentPaymentMethod(r.Context(), deps.DB, agentID)
		if err != nil {
			log.Printf("[%s] GetAgentPaymentMethod: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get payment method assignment"))
			return
		}

		if binding == nil {
			RespondJSON(w, http.StatusOK, map[string]interface{}{
				"agent_id":          agentID,
				"payment_method_id": nil,
			})
			return
		}

		RespondJSON(w, http.StatusOK, agentPaymentMethodResponse{
			AgentID:         binding.AgentID,
			PaymentMethodID: binding.PaymentMethodID,
			CreatedAt:       binding.CreatedAt,
		})
	}
}

// ── PUT /agents/{agent_id}/payment-method ───────────────────────────────────

func handleAssignAgentPaymentMethod(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		var req assignAgentPaymentMethodRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		if req.PaymentMethodID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "payment_method_id is required"))
			return
		}

		// Verify the payment method belongs to the user.
		pm, err := db.GetPaymentMethodByID(r.Context(), deps.DB, userID, req.PaymentMethodID)
		if err != nil {
			log.Printf("[%s] AssignAgentPaymentMethod: pm lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify payment method"))
			return
		}
		if pm == nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Payment method not found"))
			return
		}

		binding, err := db.AssignAgentPaymentMethod(r.Context(), deps.DB, agentID, req.PaymentMethodID)
		if err != nil {
			log.Printf("[%s] AssignAgentPaymentMethod: upsert: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to assign payment method"))
			return
		}

		RespondJSON(w, http.StatusOK, agentPaymentMethodResponse{
			AgentID:         binding.AgentID,
			PaymentMethodID: binding.PaymentMethodID,
			CreatedAt:       binding.CreatedAt,
		})
	}
}

// ── DELETE /agents/{agent_id}/payment-method ────────────────────────────────

func handleRemoveAgentPaymentMethod(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		if !requireAgentOwnership(w, r, deps, agentID, userID) {
			return
		}

		deleted, err := db.RemoveAgentPaymentMethod(r.Context(), deps.DB, agentID)
		if err != nil {
			log.Printf("[%s] RemoveAgentPaymentMethod: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to remove payment method assignment"))
			return
		}
		if !deleted {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrPaymentMethodNotFound, "No payment method assigned to this agent"))
			return
		}

		RespondJSON(w, http.StatusOK, map[string]string{"status": "removed"})
	}
}
