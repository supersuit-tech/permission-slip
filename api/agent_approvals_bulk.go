package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/shared"
)

const (
	bulkApprovalMinItems = 2
	bulkApprovalMaxItems = 50
)

type bulkApprovalRequestItem struct {
	RequestID       string                  `json:"request_id" validate:"required"`
	Action          json.RawMessage         `json:"action" validate:"required"`
	Context         json.RawMessage         `json:"context" validate:"required"`
	Configuration   *agentApprovalConfigRef `json:"configuration,omitempty"`
	PaymentMethodID string                  `json:"payment_method_id,omitempty"`
	AmountCents     *int                    `json:"amount_cents,omitempty"`
}

type agentBulkRequestApprovalRequest struct {
	Items     []bulkApprovalRequestItem `json:"items" validate:"required,min=2,max=50,dive"`
	ExpiresIn *int                      `json:"expires_in,omitempty" validate:"omitempty,gte=60,lte=86400"`
}

type bulkApprovalItemResult struct {
	RequestID          string           `json:"request_id"`
	ApprovalID         string           `json:"approval_id,omitempty"`
	Status             string           `json:"status"`
	Result             *json.RawMessage `json:"result,omitempty"`
	StandingApprovalID string           `json:"standing_approval_id,omitempty"`
	ExecutionStatus    *string          `json:"execution_status,omitempty"`
	ExecutionResult    *json.RawMessage `json:"execution_result,omitempty"`
	Error              *bulkItemError   `json:"error,omitempty"`
}

type bulkItemError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type agentBulkRequestApprovalResponse struct {
	BulkGroupID string                   `json:"bulk_group_id"`
	ApprovalURL string                   `json:"approval_url,omitempty"`
	ActionType  string                   `json:"action_type"`
	ItemCount   int                      `json:"item_count"`
	Status      string                   `json:"status"`
	ExpiresAt   *time.Time               `json:"expires_at,omitempty"`
	CreatedAt   *time.Time               `json:"created_at,omitempty"`
	Results     []bulkApprovalItemResult `json:"results"`
}

type preparedBulkItem struct {
	requestID           string
	action              json.RawMessage
	context             json.RawMessage
	actionType          string
	actionParams        json.RawMessage
	actionFingerprint   string
	connectorInstanceID string
	resourceDetails     []byte
}

func RegisterAgentBulkApprovalRoutes(mux *http.ServeMux, deps *Deps) {
	requireAgent := RequireAgentSignature(deps)
	mux.Handle("POST /approvals/bulk-request", requireAgent(handleAgentBulkRequestApproval(deps)))
	mux.Handle("GET /approval-groups/{group_id}/status", requireAgent(handleAgentBulkGroupStatus(deps)))
}

func init() {
	RegisterRouteGroup(func(mux *http.ServeMux, deps *Deps) {
		RegisterAgentBulkApprovalRoutes(mux, deps)
	})
}

func handleAgentBulkRequestApproval(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())

		var req agentBulkRequestApprovalRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		if len(req.Items) < bulkApprovalMinItems || len(req.Items) > bulkApprovalMaxItems {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
				fmt.Sprintf("bulk request must contain between %d and %d items", bulkApprovalMinItems, bulkApprovalMaxItems)))
			return
		}

		// Reject payment fields in bulk (v1).
		for i, item := range req.Items {
			if strings.TrimSpace(item.PaymentMethodID) != "" || item.AmountCents != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					fmt.Sprintf("payment actions are not supported in bulk (item %d)", i)))
				return
			}
		}

		prepared, err := prepareBulkItems(w, r, deps, agent, req.Items)
		if err != nil {
			return // response already written
		}
		if prepared == nil {
			return
		}

		actionType := prepared[0].actionType
		for _, p := range prepared[1:] {
			if p.actionType != actionType {
				errResp := BadRequest(ErrInvalidRequest, "all items in a bulk request must share the same action type")
				errResp.Error.Details = map[string]any{
					"expected_action_type":   actionType,
					"mismatched_action_type": p.actionType,
				}
				RespondError(w, r, http.StatusBadRequest, errResp)
				return
			}
		}

		if deps.Connectors != nil {
			requiresPayment, payErr := db.GetActionRequiresPaymentMethod(r.Context(), deps.DB, actionType)
			if payErr != nil {
				log.Printf("[%s] BulkRequest: payment check: %v", TraceID(r.Context()), payErr)
				CaptureError(r.Context(), payErr)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to validate action"))
				return
			}
			if requiresPayment {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					"payment-required actions are not supported in bulk"))
				return
			}
		}

		expiresAt := time.Now().UTC().Add(db.DefaultApprovalTTL)
		if req.ExpiresIn != nil {
			expiresAt = time.Now().UTC().Add(time.Duration(*req.ExpiresIn) * time.Second)
		}

		bulkGroupID, err := generatePrefixedID("bgrp_", 16)
		if err != nil {
			log.Printf("[%s] BulkRequest: generate group id: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create bulk approval"))
			return
		}

		approverProfile, err := db.GetProfileByUserID(r.Context(), deps.DB, agent.ApproverID)
		if err != nil || approverProfile == nil {
			if err != nil {
				log.Printf("[%s] BulkRequest: profile lookup: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
			}
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApproverNotFound, "Approver profile not found"))
			return
		}

		results := make([]bulkApprovalItemResult, len(prepared))
		var pendingApprovals []db.Approval
		anyPending := false

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] BulkRequest: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create bulk approval"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		group, err := db.InsertApprovalBulkGroup(r.Context(), tx, db.InsertApprovalBulkGroupParams{
			BulkGroupID: bulkGroupID,
			AgentID:     agent.AgentID,
			ApproverID:  agent.ApproverID,
			ActionType:  actionType,
			ItemCount:   len(prepared),
			ExpiresAt:   expiresAt,
		})
		if err != nil {
			log.Printf("[%s] BulkRequest: insert group: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create bulk approval"))
			return
		}

		cooldownSince := time.Now().UTC().Add(-approvalDenialCooldown(deps))

		for i, item := range prepared {
			outcome := attemptStandingApprovalForBulk(r.Context(), deps, agent, item)
			if outcome.duplicateRequest {
				RespondError(w, r, http.StatusConflict, Conflict(ErrDuplicateRequestID,
					fmt.Sprintf("A request with request_id %q has already been submitted", item.requestID)))
				return
			}
			if outcome.approved {
				results[i] = bulkApprovalItemResult{
					RequestID:          item.requestID,
					Status:             "approved",
					Result:             outcome.result,
					StandingApprovalID: outcome.standingApprovalID,
				}
				continue
			}

			if recentDenied, err := db.FindRecentDeniedApproval(r.Context(), tx, agent.AgentID, agent.ApproverID, item.actionFingerprint, cooldownSince); err != nil {
				log.Printf("[%s] BulkRequest: recent denial lookup: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create bulk approval"))
				return
			} else if recentDenied != nil {
				results[i] = bulkApprovalItemResult{
					RequestID: item.requestID,
					Status:    "denied",
					Error: &bulkItemError{
						Code:    string(ErrApprovalRecentlyDenied),
						Message: "This action was recently denied; do not retry without user intervention",
					},
				}
				continue
			}

			approvalID, genErr := generatePrefixedID("appr_", 16)
			if genErr != nil {
				log.Printf("[%s] BulkRequest: generate approval id: %v", TraceID(r.Context()), genErr)
				CaptureError(r.Context(), genErr)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create bulk approval"))
				return
			}

			groupID := bulkGroupID
			appr, insErr := db.InsertApproval(r.Context(), tx, db.InsertApprovalParams{
				ApprovalID:        approvalID,
				AgentID:           agent.AgentID,
				ApproverID:        agent.ApproverID,
				BulkGroupID:       &groupID,
				Action:            item.action,
				Context:           item.context,
				ResourceDetails:   item.resourceDetails,
				ActionFingerprint: item.actionFingerprint,
				ExpiresAt:         expiresAt,
			}, item.requestID)
			if insErr != nil {
				var apprErr *db.ApprovalError
				if errors.As(insErr, &apprErr) && apprErr.Code == db.ApprovalErrDuplicateRequest {
					RespondError(w, r, http.StatusConflict, Conflict(ErrDuplicateRequestID,
						fmt.Sprintf("A request with request_id %q has already been submitted", item.requestID)))
					return
				}
				log.Printf("[%s] BulkRequest: insert approval: %v", TraceID(r.Context()), insErr)
				CaptureError(r.Context(), insErr)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create bulk approval"))
				return
			}

			pendingApprovals = append(pendingApprovals, *appr)
			anyPending = true
			results[i] = bulkApprovalItemResult{
				RequestID:  item.requestID,
				ApprovalID: appr.ApprovalID,
				Status:     "pending",
			}
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] BulkRequest: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create bulk approval"))
				return
			}
		}

		if err := db.TouchAgentLastActive(r.Context(), deps.DB, agent.AgentID); err != nil {
			log.Printf("[%s] BulkRequest: touch last_active for agent %d: %v", TraceID(r.Context()), agent.AgentID, err)
		}

		if anyPending {
			NotifyBulkApprovalRequest(r.Context(), deps, group, agent, approverProfile, actionType)
			notifyBulkApprovalChange(deps, agent.ApproverID, bulkGroupID)
			for i := range pendingApprovals {
				emitApprovalRequestAuditEvent(r.Context(), deps.DB, agent.ApproverID, &pendingApprovals[i], agent.Metadata)
			}
		}

		groupStatus := db.BulkGroupAggregateStatus(pendingApprovals, time.Now().UTC())
		if !anyPending {
			groupStatus = "resolved"
		}

		resp := agentBulkRequestApprovalResponse{
			BulkGroupID: bulkGroupID,
			ActionType:  actionType,
			ItemCount:   len(prepared),
			Status:      groupStatus,
			ExpiresAt:   &group.ExpiresAt,
			CreatedAt:   &group.CreatedAt,
			Results:     results,
		}
		if anyPending {
			resp.ApprovalURL = fmt.Sprintf("%s/approve-group/%s", deps.BaseURL, bulkGroupID)
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

func prepareBulkItems(w http.ResponseWriter, r *http.Request, deps *Deps, agent *db.Agent, items []bulkApprovalRequestItem) ([]preparedBulkItem, error) {
	prepared := make([]preparedBulkItem, 0, len(items))
	for i, item := range items {
		item.RequestID = strings.TrimSpace(item.RequestID)
		if item.RequestID == "" || len(item.RequestID) > 255 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
				fmt.Sprintf("item %d: request_id is required and must be at most 255 characters", i)))
			return nil, fmt.Errorf("invalid request_id")
		}
		if isRawJSONNull(item.Action) || isRawJSONNull(item.Context) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
				fmt.Sprintf("item %d: action and context must be JSON objects", i)))
			return nil, fmt.Errorf("invalid json")
		}
		if err := ValidateJSONObject(item.Action); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
				fmt.Sprintf("item %d: action must be a JSON object", i)))
			return nil, fmt.Errorf("invalid json")
		}
		if err := ValidateJSONObject(item.Context); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
				fmt.Sprintf("item %d: context must be a JSON object", i)))
			return nil, fmt.Errorf("invalid json")
		}

		actionObj, actionType, ok := parseAndNormalizeBulkAction(w, r, deps, item.Action, i)
		if !ok {
			return nil, fmt.Errorf("invalid action")
		}

		connectorInstanceID, err := applyConnectorInstanceToAction(r.Context(), deps.DB, agent, actionType, actionObj)
		if err != nil {
			var ciErr *connectorInstanceResolutionError
			if errors.As(err, &ciErr) {
				RespondError(w, r, ciErr.status, ciErr.resp)
				return nil, err
			}
			log.Printf("[%s] BulkRequest prepare item %d: connector instance: %v", TraceID(r.Context()), i, err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process action"))
			return nil, err
		}
		if updated, mErr := json.Marshal(actionObj); mErr == nil {
			item.Action = updated
		}

		actionParams := json.RawMessage(actionObj["parameters"])
		if !validateActionParameters(w, r, deps.DB, actionType, actionParams) {
			return nil, fmt.Errorf("invalid parameters")
		}

		if item.Configuration != nil {
			result := ValidateConfigurationReference(w, r, deps, item.Configuration.ConfigurationID, agent.AgentID, actionType, actionParams)
			if result == nil {
				return nil, fmt.Errorf("invalid configuration")
			}
		}

		resourceDetails := resolveResourceDetailsForBulk(r.Context(), deps, agent, actionType, actionParams, connectorInstanceID)
		contextForStore := mergeSlackContextFromResourceDetailsIntoContext(
			mergeEmailThreadFromResourceDetailsIntoContext(item.Context, resourceDetails),
			resourceDetails,
		)

		prepared = append(prepared, preparedBulkItem{
			requestID:           item.RequestID,
			action:              item.Action,
			context:             contextForStore,
			actionType:          actionType,
			actionParams:        actionParams,
			actionFingerprint:   db.ComputeActionFingerprint(agent.AgentID, agent.ApproverID, item.Action),
			connectorInstanceID: connectorInstanceID,
			resourceDetails:     resourceDetails,
		})
	}
	return prepared, nil
}

func parseAndNormalizeBulkAction(w http.ResponseWriter, r *http.Request, deps *Deps, action json.RawMessage, index int) (map[string]json.RawMessage, string, bool) {
	var actionObj map[string]json.RawMessage
	if err := json.Unmarshal(action, &actionObj); err != nil {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
			fmt.Sprintf("item %d: action must be a JSON object", index)))
		return nil, "", false
	}
	typeRaw, hasType := actionObj["type"]
	if !hasType {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
			fmt.Sprintf("item %d: action.type is required", index)))
		return nil, "", false
	}
	var actionType string
	if err := json.Unmarshal(typeRaw, &actionType); err != nil || strings.TrimSpace(actionType) == "" {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
			fmt.Sprintf("item %d: action.type must be a non-empty string", index)))
		return nil, "", false
	}
	if len(actionType) > shared.ActionTypeMaxLength {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
			fmt.Sprintf("item %d: action.type exceeds maximum length", index)))
		return nil, "", false
	}

	if deps.Connectors != nil {
		connAction, conn, ok := deps.Connectors.GetActionWithConnector(actionType)
		if !ok {
			errResp := BadRequest(ErrUnsupportedActionType, fmt.Sprintf("unknown action type %q", actionType))
			errResp.Error.Details = map[string]any{"action_type": actionType, "item_index": index}
			RespondError(w, r, http.StatusBadRequest, errResp)
			return nil, "", false
		}
		if aliaser, ok := connAction.(connectors.ParameterAliaser); ok {
			if aliases := aliaser.ParameterAliases(); len(aliases) > 0 {
				if rawParams, hasParams := actionObj["parameters"]; hasParams {
					actionObj["parameters"] = connectors.NormalizeParameters(aliases, rawParams)
				}
			}
		}
		if normalizer, ok := connAction.(connectors.Normalizer); ok {
			if rawParams, hasParams := actionObj["parameters"]; hasParams {
				actionObj["parameters"] = normalizer.Normalize(rawParams)
			}
		}
		if rawParams, hasParams := actionObj["parameters"]; hasParams {
			var validationErr error
			if rv, ok := connAction.(connectors.RequestValidator); ok {
				validationErr = rv.ValidateRequest(json.RawMessage(rawParams))
			} else if pv, ok := conn.(connectors.ParamValidator); ok {
				validationErr = pv.ValidateParams(actionType, json.RawMessage(rawParams))
			}
			if validationErr != nil {
				var ve *connectors.ValidationError
				msg := "invalid action parameters"
				if errors.As(validationErr, &ve) {
					msg = ve.Message
				}
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					fmt.Sprintf("item %d: %s", index, msg)))
				return nil, "", false
			}
		}
	}

	return actionObj, actionType, true
}

func resolveResourceDetailsForBulk(ctx context.Context, deps *Deps, agent *db.Agent, actionType string, actionParams json.RawMessage, connectorInstanceID string) []byte {
	if deps.Connectors == nil {
		return nil
	}
	cid := strings.SplitN(actionType, ".", 2)
	if len(cid) != 2 {
		return nil
	}
	conn, ok := deps.Connectors.Get(cid[0])
	if !ok {
		return nil
	}
	resolver, ok := conn.(connectors.ResourceDetailResolver)
	if !ok {
		return nil
	}
	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	details, err := resolver.ResolveResourceDetails(resolveCtx, actionType, actionParams,
		resolveCredentialsForResolver(resolveCtx, deps, agent.AgentID, agent.ApproverID, actionType, cid[0], connectorInstanceID))
	if err != nil {
		log.Printf("[%s] BulkRequest ResolveResourceDetails: %v", TraceID(ctx), err)
		return nil
	}
	if details == nil {
		return nil
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return nil
	}
	return encoded
}

type bulkStandingOutcome struct {
	approved           bool
	duplicateRequest   bool
	result             *json.RawMessage
	standingApprovalID string
}

func attemptStandingApprovalForBulk(ctx context.Context, deps *Deps, agent *db.Agent, item preparedBulkItem) bulkStandingOutcome {
	approvals, err := db.FindActiveStandingApprovalsForAgent(ctx, deps.DB, agent.AgentID, item.actionType, item.connectorInstanceID)
	if err != nil || len(approvals) == 0 {
		return bulkStandingOutcome{}
	}

	var sa *db.StandingApproval
	for _, candidate := range approvals {
		if len(candidate.Constraints) == 0 {
			sa = candidate
			break
		}
		if err := db.ValidateParametersAgainstConfig(candidate.Constraints, item.actionParams); err != nil {
			continue
		}
		sa = candidate
		break
	}
	if sa == nil {
		return bulkStandingOutcome{}
	}

	exec, err := db.RecordStandingApprovalExecutionByAgent(ctx, deps.DB, sa.StandingApprovalID, agent.AgentID, item.requestID, item.actionParams)
	if err != nil {
		var saErr *db.StandingApprovalError
		if errors.As(err, &saErr) && saErr.Code == db.StandingApprovalErrDuplicateRequest {
			return bulkStandingOutcome{duplicateRequest: true}
		}
		return bulkStandingOutcome{}
	}

	result, execErr := executeConnectorAction(ctx, deps, exec.AgentID, exec.UserID, item.actionType, item.actionParams, nil, item.connectorInstanceID)
	emitStandingApprovalAuditEvent(ctx, deps.DB, exec.UserID, exec.AgentID, sa.StandingApprovalID, exec.ActionType, exec.AgentMeta, execErr)
	if execErr != nil {
		return bulkStandingOutcome{}
	}

	NotifyStandingApprovalExecution(ctx, deps, exec, agent, item.actionType, item.actionParams)

	var actionResultPtr *json.RawMessage
	if result != nil {
		actionResultPtr = &result.Data
	}

	return bulkStandingOutcome{
		approved:           true,
		result:             actionResultPtr,
		standingApprovalID: sa.StandingApprovalID,
	}
}

func handleAgentBulkGroupStatus(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		groupID := strings.TrimSpace(r.PathValue("group_id"))
		if groupID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "group_id is required"))
			return
		}

		group, err := db.GetApprovalBulkGroupByIDAndAgent(r.Context(), deps.DB, groupID, agent.AgentID)
		if err != nil {
			log.Printf("[%s] BulkGroupStatus: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get bulk group status"))
			return
		}
		if group == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrApprovalNotFound, "Bulk group not found"))
			return
		}

		approvals, err := db.ListApprovalsByBulkGroupID(r.Context(), deps.DB, groupID)
		if err != nil {
			log.Printf("[%s] BulkGroupStatus list: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get bulk group status"))
			return
		}

		RespondJSON(w, http.StatusOK, buildBulkGroupStatusResponse(*group, approvals))
	}
}

func buildBulkGroupStatusResponse(group db.ApprovalBulkGroup, approvals []db.Approval) agentBulkGroupStatusResponse {
	results := make([]bulkApprovalItemResult, 0, len(approvals))
	for _, a := range approvals {
		status := resolvedApprovalStatus(a)
		item := bulkApprovalItemResult{
			RequestID:  "",
			ApprovalID: a.ApprovalID,
			Status:     status,
		}
		if a.ExecutionStatus != nil {
			item.ExecutionStatus = a.ExecutionStatus
		}
		if len(a.ExecutionResult) > 0 {
			raw := json.RawMessage(a.ExecutionResult)
			item.ExecutionResult = &raw
		}
		results = append(results, item)
	}

	return agentBulkGroupStatusResponse{
		BulkGroupID: group.BulkGroupID,
		ActionType:  group.ActionType,
		ItemCount:   group.ItemCount,
		Status:      db.BulkGroupAggregateStatus(approvals, time.Now().UTC()),
		ExpiresAt:   group.ExpiresAt,
		CreatedAt:   group.CreatedAt,
		Results:     results,
	}
}

type agentBulkGroupStatusResponse struct {
	BulkGroupID string                   `json:"bulk_group_id"`
	ActionType  string                   `json:"action_type"`
	ItemCount   int                      `json:"item_count"`
	Status      string                   `json:"status"`
	ExpiresAt   time.Time                `json:"expires_at"`
	CreatedAt   time.Time                `json:"created_at"`
	Results     []bulkApprovalItemResult `json:"results"`
}
