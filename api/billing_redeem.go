package api

import (
	"log"
	"net/http"
	"strings"

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
		couponHex := strings.TrimSpace(req.Coupon)
		if couponHex == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "coupon is required"))
			return
		}
		if !coupon.ValidFreeProCouponHexFormat(strings.ToLower(couponHex)) {
			RespondError(w, r, http.StatusForbidden, Forbidden(ErrInvalidCoupon, "Invalid coupon"))
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
		if sub.PlanID != db.PlanFree && sub.PlanID != db.PlanPayAsYouGo {
			RespondError(w, r, http.StatusConflict, Conflict(ErrPlanChangeNotAllowed, "Coupon can only be applied on the free or pay-as-you-go plan"))
			return
		}

		if !coupon.FreeProCouponMatches(profile.Username, deps.CouponSecret, couponHex) {
			RespondError(w, r, http.StatusForbidden, Forbidden(ErrInvalidCoupon, "Invalid coupon"))
			return
		}

		// Paid plan: cancel Stripe first so we do not leave an active paid subscription
		// while the app treats the user as complimentary Pro.
		if sub.PlanID == db.PlanPayAsYouGo && sub.StripeSubscriptionID != nil {
			if deps.Stripe == nil {
				RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Billing provider not configured"))
				return
			}
			if _, err := deps.Stripe.CancelSubscription(r.Context(), *sub.StripeSubscriptionID); err != nil {
				log.Printf("[%s] RedeemCoupon: cancel Stripe subscription: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusBadGateway, upstreamError("Failed to cancel subscription with payment provider"))
				return
			}
		}

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] RedeemCoupon: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply coupon"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		cur, err := db.GetSubscriptionByUserID(r.Context(), tx, profile.ID)
		if err != nil {
			log.Printf("[%s] RedeemCoupon: reload subscription: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply coupon"))
			return
		}
		if cur == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrSubscriptionNotFound, "No subscription found"))
			return
		}
		if cur.PlanID == db.PlanFreePro {
			RespondJSON(w, http.StatusOK, redeemCouponResponse{PlanID: db.PlanFreePro, Status: "already_redeemed"})
			return
		}

		var updated *db.Subscription
		switch cur.PlanID {
		case db.PlanPayAsYouGo:
			var custPtr *string
			if cur.StripeCustomerID != nil {
				custPtr = cur.StripeCustomerID
			}
			if _, err := db.UpdateSubscriptionStripe(r.Context(), tx, profile.ID, custPtr, nil); err != nil {
				log.Printf("[%s] RedeemCoupon: clear stripe subscription id: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply coupon"))
				return
			}
			updated, err = db.UpgradeSubscriptionPlan(r.Context(), tx, profile.ID, db.PlanPayAsYouGo, db.PlanFreePro)
		case db.PlanFree:
			updated, err = db.UpgradeSubscriptionPlan(r.Context(), tx, profile.ID, db.PlanFree, db.PlanFreePro)
		default:
			RespondError(w, r, http.StatusConflict, Conflict(ErrPlanChangeNotAllowed, "Subscription changed while redeeming coupon"))
			return
		}
		if err != nil {
			log.Printf("[%s] RedeemCoupon: upgrade plan: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply coupon"))
			return
		}
		if updated == nil {
			again, err := db.GetSubscriptionByUserID(r.Context(), tx, profile.ID)
			if err != nil {
				log.Printf("[%s] RedeemCoupon: reload after no-op upgrade: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply coupon"))
				return
			}
			if again != nil && again.PlanID == db.PlanFreePro {
				if owned {
					if err := db.CommitTx(r.Context(), tx); err != nil {
						log.Printf("[%s] RedeemCoupon: commit: %v", TraceID(r.Context()), err)
						CaptureError(r.Context(), err)
						RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply coupon"))
						return
					}
				}
				RespondJSON(w, http.StatusOK, redeemCouponResponse{PlanID: db.PlanFreePro, Status: "already_redeemed"})
				return
			}
			RespondError(w, r, http.StatusConflict, Conflict(ErrPlanChangeNotAllowed, "Subscription changed while redeeming coupon"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] RedeemCoupon: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply coupon"))
				return
			}
		}

		log.Printf("[%s] RedeemCoupon: user %s upgraded to free_pro", TraceID(r.Context()), profile.ID)
		RespondJSON(w, http.StatusOK, redeemCouponResponse{PlanID: db.PlanFreePro, Status: "redeemed"})
	}
}
