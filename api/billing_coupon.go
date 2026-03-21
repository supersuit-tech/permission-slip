package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ── POST /billing/redeem-coupon ────────────────────────────────────────────

type redeemCouponRequest struct {
	CouponCode string `json:"coupon_code"`
}

type redeemCouponResponse struct {
	Status string `json:"status"`
	PlanID string `json:"plan_id"`
}

func init() {
	RegisterRouteGroup(RegisterCouponRoutes)
}

func RegisterCouponRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("POST /billing/redeem-coupon", requireProfile(handleRedeemCoupon(deps)))
}

func handleRedeemCoupon(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req redeemCouponRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid request body"))
			return
		}

		if strings.TrimSpace(req.CouponCode) == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Coupon code is required"))
			return
		}

		// Check if user already has free pro.
		sub, err := db.GetSubscriptionByUserID(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] RedeemCoupon: subscription lookup: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to fetch subscription"))
			return
		}
		if sub == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrSubscriptionNotFound, "No subscription found"))
			return
		}
		if sub.IsFreePro {
			// Already a free pro user — idempotent success.
			RespondJSON(w, http.StatusOK, redeemCouponResponse{
				Status: "already_redeemed",
				PlanID: sub.PlanID,
			})
			return
		}

		// Validate the coupon code: MD5(username + "-" + COUPON_SECRET)
		couponSecret := os.Getenv("COUPON_SECRET")
		if couponSecret == "" {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Coupon redemption is not available"))
			return
		}

		expected := md5Hash(profile.Username + "-" + couponSecret)
		if !strings.EqualFold(strings.TrimSpace(req.CouponCode), expected) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidCode, "Invalid coupon code"))
			return
		}

		// Grant free pro.
		updated, err := db.GrantFreePro(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] RedeemCoupon: grant free pro: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to activate free pro"))
			return
		}

		RespondJSON(w, http.StatusOK, redeemCouponResponse{
			Status: "redeemed",
			PlanID: updated.PlanID,
		})
	}
}

func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
