package api

import (
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/coupon"
	"github.com/supersuit-tech/permission-slip-web/db"
)

type redeemCouponRequest struct {
	Coupon string `json:"coupon"`
}

type redeemCouponResponse struct {
	PlanID string `json:"plan_id"`
	Status string `json:"status"`
}

// RegisterBillingRedeemRoute registers POST /billing/redeem-coupon when billing
// is enabled and COUPON_SECRET is non-empty.
func RegisterBillingRedeemRoute(mux *http.ServeMux, deps *Deps) {
	if !deps.BillingEnabled || deps.CouponSecret == "" {
		return
	}
	requireProfile := RequireProfile(deps)
	mux.Handle("POST /billing/redeem-coupon", requireProfile(handleRedeemFreeProCoupon(deps)))
}

func handleRedeemFreeProCoupon(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req redeemCouponRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if req.Coupon == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "coupon is required"))
			return
		}

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

		if sub.PlanID == db.PlanFreePro {
			RespondJSON(w, http.StatusOK, redeemCouponResponse{PlanID: db.PlanFreePro, Status: "already_redeemed"})
			return
		}
		if sub.PlanID == db.PlanPayAsYouGo {
			RespondError(w, r, http.StatusConflict, Conflict(ErrAlreadySubscribed, "Paid subscription already active"))
			return
		}
		if sub.PlanID != db.PlanFree {
			RespondError(w, r, http.StatusConflict, Conflict(ErrPlanChangeNotAllowed, "Coupon can only be applied on the free plan"))
			return
		}

		if !coupon.FreeProCouponMatches(profile.Username, deps.CouponSecret, req.Coupon) {
			RespondError(w, r, http.StatusForbidden, Forbidden(ErrInvalidCoupon, "Invalid coupon"))
			return
		}

		updated, err := db.UpgradeSubscriptionPlan(r.Context(), deps.DB, profile.ID, db.PlanFree, db.PlanFreePro)
		if err != nil {
			log.Printf("[%s] RedeemCoupon: upgrade plan: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply coupon"))
			return
		}
		if updated == nil {
			RespondJSON(w, http.StatusOK, redeemCouponResponse{PlanID: db.PlanFreePro, Status: "already_redeemed"})
			return
		}

		log.Printf("[%s] RedeemCoupon: user %s upgraded to free_pro", TraceID(r.Context()), profile.ID)
		RespondJSON(w, http.StatusOK, redeemCouponResponse{PlanID: db.PlanFreePro, Status: "redeemed"})
	}
}
