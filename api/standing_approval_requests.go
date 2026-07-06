package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

type standingApprovalRequestResponse struct {
	RequestID                   string     `json:"request_id"`
	AgentID                     int64      `json:"agent_id"`
	UserID                      string     `json:"user_id"`
	ActionType                  string     `json:"action_type"`
	ActionVersion               string     `json:"action_version"`
	Constraints                 any        `json:"constraints"`
	ConnectorName               *string    `json:"connector_name,omitempty"`
	ConnectorInstanceID         *string    `json:"connector_instance_id,omitempty"`
	ConnectorInstanceDisplay    *string    `json:"connector_instance_display,omitempty"`
	Status                      string     `json:"status"`
	DecidedAt                   *time.Time `json:"decided_at,omitempty"`
	ResultingStandingApprovalID *string    `json:"resulting_standing_approval_id,omitempty"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}

type standingApprovalRequestListResponse struct {
	Data       []standingApprovalRequestResponse `json:"data"`
	HasMore    bool                              `json:"has_more"`
	NextCursor *string                           `json:"next_cursor,omitempty"`
}

type approveStandingApprovalRequestResponse struct {
	RequestID                   string                    `json:"request_id"`
	Status                      string                    `json:"status"`
	ResultingStandingApprovalID string                    `json:"resulting_standing_approval_id"`
	StandingApproval            *standingApprovalResponse `json:"standing_approval,omitempty"`
}

type denyStandingApprovalRequestResponse struct {
	RequestID string    `json:"request_id"`
	Status    string    `json:"status"`
	DecidedAt time.Time `json:"decided_at"`
}

var validStandingApprovalRequestStatusFilters = map[string]bool{
	"pending":  true,
	"approved": true,
	"denied":   true,
	"all":      true,
}

func init() {
	RegisterRouteGroup(RegisterStandingApprovalRequestRoutes)
}

func RegisterStandingApprovalRequestRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /standing-approval-requests", requireProfile(handleListStandingApprovalRequests(deps)))
	mux.Handle("GET /standing-approval-requests/{request_id}", requireProfile(handleGetStandingApprovalRequest(deps)))
	mux.Handle("POST /standing-approval-requests/{request_id}/approve", requireProfile(handleApproveStandingApprovalRequest(deps)))
	mux.Handle("POST /standing-approval-requests/{request_id}/deny", requireProfile(handleDenyStandingApprovalRequest(deps)))
}

func toStandingApprovalRequestResponse(sar db.StandingApprovalRequest) standingApprovalRequestResponse {
	var constraints any
	if len(sar.Constraints) > 0 {
		_ = json.Unmarshal(sar.Constraints, &constraints)
	}
	return standingApprovalRequestResponse{
		RequestID:                   sar.RequestID,
		AgentID:                     sar.AgentID,
		UserID:                      sar.UserID,
		ActionType:                  sar.ActionType,
		ActionVersion:               sar.ActionVersion,
		Constraints:                 constraints,
		ConnectorName:               sar.ConnectorName,
		ConnectorInstanceID:         sar.ConnectorInstanceID,
		ConnectorInstanceDisplay:    sar.ConnectorInstanceDisplay,
		Status:                      sar.Status,
		DecidedAt:                   sar.DecidedAt,
		ResultingStandingApprovalID: sar.ResultingStandingApprovalID,
		CreatedAt:                   sar.CreatedAt,
		UpdatedAt:                   sar.UpdatedAt,
	}
}

func handleListStandingApprovalRequests(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		statusFilter := r.URL.Query().Get("status")
		if statusFilter == "" {
			statusFilter = "pending"
		}
		if !validStandingApprovalRequestStatusFilters[statusFilter] {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid status filter; must be one of: pending, approved, denied, all"))
			return
		}

		limit, ok := parsePaginationLimit(w, r)
		if !ok {
			return
		}

		var cursor *db.StandingApprovalRequestCursor
		if v := r.URL.Query().Get("after"); v != "" {
			c, err := parseStandingApprovalRequestCursor(v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid pagination cursor"))
				return
			}
			cursor = c
		}

		page, err := db.ListStandingApprovalRequestsByUser(r.Context(), deps.DB, profile.ID, statusFilter, limit, cursor)
		if err != nil {
			log.Printf("[%s] ListStandingApprovalRequests: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list standing approval requests"))
			return
		}

		data := make([]standingApprovalRequestResponse, len(page.Requests))
		for i, sar := range page.Requests {
			data[i] = toStandingApprovalRequestResponse(sar)
		}

		resp := standingApprovalRequestListResponse{Data: data, HasMore: page.HasMore}
		if page.HasMore && len(page.Requests) > 0 {
			last := page.Requests[len(page.Requests)-1]
			c := encodeStandingApprovalRequestCursor(last)
			resp.NextCursor = &c
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleGetStandingApprovalRequest(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		requestID := strings.TrimSpace(r.PathValue("request_id"))
		if requestID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "request_id is required"))
			return
		}

		sar, err := db.GetStandingApprovalRequestByIDAndUser(r.Context(), deps.DB, requestID, profile.ID)
		if err != nil {
			log.Printf("[%s] GetStandingApprovalRequest: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get standing approval request"))
			return
		}
		if sar == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Standing approval request not found"))
			return
		}

		RespondJSON(w, http.StatusOK, toStandingApprovalRequestResponse(*sar))
	}
}

func handleApproveStandingApprovalRequest(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		requestID := strings.TrimSpace(r.PathValue("request_id"))
		if requestID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "request_id is required"))
			return
		}

		sar, err := db.GetStandingApprovalRequestByIDAndUser(r.Context(), deps.DB, requestID, profile.ID)
		if err != nil {
			log.Printf("[%s] ApproveStandingApprovalRequest load: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to approve standing approval request"))
			return
		}
		if sar == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Standing approval request not found"))
			return
		}

		if sar.Status == db.StandingApprovalRequestStatusApproved && sar.ResultingStandingApprovalID != nil {
			sa, _ := db.GetStandingApprovalByIDAndUser(r.Context(), deps.DB, *sar.ResultingStandingApprovalID, profile.ID)
			resp := approveStandingApprovalRequestResponse{
				RequestID:                   sar.RequestID,
				Status:                      sar.Status,
				ResultingStandingApprovalID: *sar.ResultingStandingApprovalID,
			}
			if sa != nil {
				saResp := toStandingApprovalResponse(*sa)
				resp.StandingApproval = &saResp
			}
			RespondJSON(w, http.StatusOK, resp)
			return
		}
		if sar.Status != db.StandingApprovalRequestStatusPending {
			RespondError(w, r, http.StatusConflict, Conflict(ErrApprovalAlreadyResolved, "Standing approval request is no longer pending"))
			return
		}

		startsAt := time.Now().UTC()
		ruleName := autoCreatedFromRuleProposalName

		actionSchema, err := db.GetActionParametersSchema(r.Context(), tx, sar.ActionType)
		if err != nil {
			log.Printf("[%s] ApproveStandingApprovalRequest lookup action: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to approve standing approval request"))
			return
		}
		if actionSchema == nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "action type is not registered for any connector"))
			return
		}

		saID, err := generatePrefixedID("sa_", 16)
		if err != nil {
			log.Printf("[%s] ApproveStandingApprovalRequest generate SA id: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to approve standing approval request"))
			return
		}

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] ApproveStandingApprovalRequest begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to approve standing approval request"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		sa, err := db.CreateStandingApproval(r.Context(), tx, db.CreateStandingApprovalParams{
			StandingApprovalID:  saID,
			AgentID:             sar.AgentID,
			UserID:              profile.ID,
			ActionType:          sar.ActionType,
			ActionVersion:       sar.ActionVersion,
			Constraints:         sar.Constraints,
			Name:                &ruleName,
			Description:         &autoCreatedFromRuleProposalDescription,
			ConnectorInstanceID: sar.ConnectorInstanceID,
			StartsAt:            startsAt,
			ExpiresAt:           nil,
		})
		if err != nil {
			if handleStandingApprovalError(w, r, err) {
				return
			}
			log.Printf("[%s] CreateStandingApproval from request: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to approve standing approval request"))
			return
		}

		updated, err := db.ApproveStandingApprovalRequest(r.Context(), tx, db.ApproveStandingApprovalRequestParams{
			RequestID:          requestID,
			UserID:             profile.ID,
			StandingApprovalID: saID,
		})
		if err != nil {
			var reqErr *db.StandingApprovalRequestError
			if errors.As(err, &reqErr) {
				handleStandingApprovalRequestError(w, r, err)
				return
			}
			log.Printf("[%s] ApproveStandingApprovalRequest update: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to approve standing approval request"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] ApproveStandingApprovalRequest commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to approve standing approval request"))
				return
			}
		}

		saResp := toStandingApprovalResponse(*sa)
		RespondJSON(w, http.StatusOK, approveStandingApprovalRequestResponse{
			RequestID:                   updated.RequestID,
			Status:                      updated.Status,
			ResultingStandingApprovalID: saID,
			StandingApproval:            &saResp,
		})
	}
}

func handleDenyStandingApprovalRequest(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		requestID := strings.TrimSpace(r.PathValue("request_id"))
		if requestID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "request_id is required"))
			return
		}

		sar, err := db.DenyStandingApprovalRequest(r.Context(), deps.DB, requestID, profile.ID)
		if err != nil {
			if handleStandingApprovalRequestError(w, r, err) {
				return
			}
			log.Printf("[%s] DenyStandingApprovalRequest: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to deny standing approval request"))
			return
		}

		decidedAt := time.Now().UTC()
		if sar.DecidedAt != nil {
			decidedAt = *sar.DecidedAt
		}

		RespondJSON(w, http.StatusOK, denyStandingApprovalRequestResponse{
			RequestID: sar.RequestID,
			Status:    sar.Status,
			DecidedAt: decidedAt,
		})
	}
}

func handleStandingApprovalRequestError(w http.ResponseWriter, r *http.Request, err error) bool {
	var reqErr *db.StandingApprovalRequestError
	if !errors.As(err, &reqErr) {
		return false
	}
	switch reqErr.Code {
	case db.StandingApprovalRequestErrNotFound:
		RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Standing approval request not found"))
	case db.StandingApprovalRequestErrAlreadyResolved:
		RespondError(w, r, http.StatusConflict, Conflict(ErrApprovalAlreadyResolved, "Standing approval request is no longer pending"))
	default:
		RespondError(w, r, http.StatusForbidden, Forbidden(ErrAgentNotAuthorized, "Not allowed to modify this standing approval request"))
	}
	return true
}

func parseStandingApprovalRequestCursor(v string) (*db.StandingApprovalRequestCursor, error) {
	parts := strings.SplitN(v, ",", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, err
	}
	return &db.StandingApprovalRequestCursor{CreatedAt: ts, RequestID: parts[1]}, nil
}

func encodeStandingApprovalRequestCursor(sar db.StandingApprovalRequest) string {
	return sar.CreatedAt.UTC().Format(time.RFC3339Nano) + "," + sar.RequestID
}
