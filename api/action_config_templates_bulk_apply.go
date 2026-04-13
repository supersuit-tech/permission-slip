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

	"github.com/jackc/pgx/v5"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/shared"
)

// --- Request / Response types ---

type bulkApplyActionConfigTemplateRequest struct {
	AgentID     int64    `json:"agent_id" validate:"gt=0"`
	TemplateIDs []string `json:"template_ids" validate:"required,min=1,max=50,dive,required,max=255"`
}

type bulkApplyResultError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type bulkApplyResult struct {
	TemplateID          string                        `json:"template_id"`
	Success             bool                          `json:"success"`
	ActionConfiguration *actionConfigResponse         `json:"action_configuration,omitempty"`
	StandingApproval    *standingApprovalResponse     `json:"standing_approval,omitempty"`
	Error               *bulkApplyResultError         `json:"error,omitempty"`
}

type bulkApplyActionConfigTemplateResponse struct {
	Results []bulkApplyResult `json:"results"`
}

// --- Handler ---

func handleBulkApplyActionConfigTemplates(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req bulkApplyActionConfigTemplateRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		// Deduplicate template IDs while preserving order.
		uniqueIDs := deduplicateStrings(req.TemplateIDs)

		// Fetch all templates in one query.
		templates, err := db.GetActionConfigTemplatesByIDs(r.Context(), deps.DB, uniqueIDs)
		if err != nil {
			log.Printf("[%s] BulkApplyTemplates: fetch templates: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
			return
		}

		// Build lookup map and check for missing templates.
		tplByID := make(map[string]*db.ActionConfigTemplate, len(templates))
		for i := range templates {
			tplByID[templates[i].ID] = &templates[i]
		}
		for _, id := range uniqueIDs {
			if _, ok := tplByID[id]; !ok {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrActionConfigTemplateNotFound,
					fmt.Sprintf("Template %q not found", id)))
				return
			}
		}

		// Verify all templates belong to the same connector.
		var connectorID string
		for _, id := range uniqueIDs {
			tpl := tplByID[id]
			if connectorID == "" {
				connectorID = tpl.ConnectorID
			} else if tpl.ConnectorID != connectorID {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest,
					"All templates must belong to the same connector"))
				return
			}
		}

		// Begin transaction.
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] BulkApplyTemplates: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		// Enable connector once for the agent.
		enabled, err := db.AgentConnectorEnabled(r.Context(), tx, req.AgentID, profile.ID, connectorID)
		if err != nil {
			log.Printf("[%s] BulkApplyTemplates: connector check: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
			return
		}
		if !enabled {
			if _, err := db.EnableAgentConnector(r.Context(), tx, req.AgentID, profile.ID, connectorID); err != nil {
				var acErr *db.AgentConnectorError
				if errors.As(err, &acErr) && acErr.Code == db.AgentConnectorErrAgentNotFound {
					RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
					return
				}
				if errors.As(err, &acErr) && acErr.Code == db.AgentConnectorErrConnectorNotFound {
					RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidReference, "Invalid connector reference"))
					return
				}
				log.Printf("[%s] BulkApplyTemplates: enable connector: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
				return
			}
		}

		// Check if any template needs a standing approval — acquire the lock once.
		anyWantSA := false
		for _, id := range uniqueIDs {
			tpl := tplByID[id]
			if len(tpl.StandingApprovalSpec) > 0 && string(tpl.StandingApprovalSpec) != "null" {
				anyWantSA = true
				break
			}
		}
		if anyWantSA {
			if err := db.AcquireStandingApprovalLimitLock(r.Context(), tx, profile.ID); err != nil {
				log.Printf("[%s] BulkApplyTemplates: advisory lock: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
				return
			}
		}

		// Apply each template. Use savepoints so individual failures don't
		// abort the entire batch.
		pgxTx, _ := tx.(pgx.Tx)
		results := make([]bulkApplyResult, 0, len(uniqueIDs))

		for _, id := range uniqueIDs {
			tpl := tplByID[id]
			res, err := applyOneTemplateInSavepoint(r.Context(), pgxTx, tx, profile, tpl, req.AgentID)
			if err != nil {
				results = append(results, bulkApplyResult{
					TemplateID: id,
					Success:    false,
					Error:      &bulkApplyResultError{Code: res.errorCode, Message: err.Error()},
				})
				continue
			}

			br := bulkApplyResult{
				TemplateID:          id,
				Success:             true,
				ActionConfiguration: res.actionConfig,
			}
			if res.standingApproval != nil {
				br.StandingApproval = res.standingApproval
			}
			results = append(results, br)
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] BulkApplyTemplates: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, bulkApplyActionConfigTemplateResponse{Results: results})
	}
}

// --- Internal helpers ---

type applyOneResult struct {
	actionConfig     *actionConfigResponse
	standingApproval *standingApprovalResponse
	errorCode        string
}

// applyOneTemplateInSavepoint applies a single template within a savepoint.
// If pgxTx is non-nil, savepoints are used for isolation. Otherwise, the
// operation runs directly on fallbackTx (e.g., in tests where DBTX is already a tx).
func applyOneTemplateInSavepoint(
	ctx context.Context,
	pgxTx pgx.Tx,
	fallbackTx db.DBTX,
	profile *db.Profile,
	tpl *db.ActionConfigTemplate,
	agentID int64,
) (*applyOneResult, error) {
	var dtx db.DBTX
	var sp pgx.Tx

	if pgxTx != nil {
		var err error
		sp, err = pgxTx.Begin(ctx)
		if err != nil {
			return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("internal error")
		}
		defer sp.Rollback(ctx) //nolint:errcheck
		dtx = sp
	} else {
		dtx = fallbackTx
	}

	res, err := applyOneTemplateCore(ctx, dtx, profile, tpl, agentID)
	if err != nil {
		return res, err
	}

	if sp != nil {
		if err := sp.Commit(ctx); err != nil {
			return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("internal error")
		}
	}

	return res, nil
}

// applyOneTemplateCore contains the validation and creation logic for a
// single template, without transaction management.
func applyOneTemplateCore(
	ctx context.Context,
	tx db.DBTX,
	profile *db.Profile,
	tpl *db.ActionConfigTemplate,
	agentID int64,
) (*applyOneResult, error) {
	// Validate template fields.
	if len(tpl.Name) > shared.ActionConfigNameMaxLength {
		return &applyOneResult{errorCode: string(ErrInvalidRequest)}, fmt.Errorf("template name exceeds maximum length")
	}
	if tpl.Description != nil && len(*tpl.Description) > shared.ActionConfigDescMaxLength {
		return &applyOneResult{errorCode: string(ErrInvalidRequest)}, fmt.Errorf("template description exceeds maximum length")
	}

	params := tpl.Parameters
	if len(params) == 0 {
		params = []byte("{}")
	} else if err := ValidateJSONObject(params); err != nil {
		return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("invalid template parameters")
	}

	actionType := strings.TrimSpace(tpl.ActionType)
	if actionType != db.WildcardActionType && strings.Contains(actionType, "*") {
		return &applyOneResult{errorCode: string(ErrInvalidRequest)}, fmt.Errorf("invalid template action_type")
	}

	if actionType != db.WildcardActionType {
		if err := db.ValidateConfigParameters(params); err != nil {
			return &applyOneResult{errorCode: string(ErrInvalidRequest)}, fmt.Errorf("%s", err.Error())
		}
		exists, err := db.ConnectorActionExists(ctx, tx, tpl.ConnectorID, actionType)
		if err != nil {
			return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("internal error")
		}
		if !exists {
			return &applyOneResult{errorCode: string(ErrInvalidReference)}, fmt.Errorf("invalid connector or action type reference")
		}
	}

	configID, err := generatePrefixedID("ac_", 16)
	if err != nil {
		return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("internal error")
	}

	// Parse standing approval spec before creating the config.
	var spec standingApprovalTemplateSpec
	var standingBytes []byte
	wantStanding := len(tpl.StandingApprovalSpec) > 0 && string(tpl.StandingApprovalSpec) != "null"
	if wantStanding {
		if err := json.Unmarshal(tpl.StandingApprovalSpec, &spec); err != nil {
			return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("invalid standing approval spec")
		}
		standingBytes, err = buildStandingApprovalConstraintsFromTemplate(params)
		if err != nil {
			return &applyOneResult{errorCode: string(ErrInvalidRequest)}, fmt.Errorf("%s", err.Error())
		}
	}

	ac, err := db.CreateActionConfig(ctx, tx, db.CreateActionConfigParams{
		ID:          configID,
		AgentID:     agentID,
		UserID:      profile.ID,
		ConnectorID: tpl.ConnectorID,
		ActionType:  actionType,
		Parameters:  params,
		Name:        tpl.Name,
		Description: tpl.Description,
	})
	if err != nil {
		var acErr *db.ActionConfigError
		if errors.As(err, &acErr) {
			switch acErr.Code {
			case db.ActionConfigErrAgentNotFound:
				return &applyOneResult{errorCode: string(ErrAgentNotFound)}, fmt.Errorf("agent not found or not owned by user")
			case db.ActionConfigErrInvalidRef:
				return &applyOneResult{errorCode: string(ErrInvalidReference)}, fmt.Errorf("invalid connector, action type, or credential reference")
			}
		}
		return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("internal error")
	}

	var saOut *db.StandingApproval
	if wantStanding {
		// Check standing approval limit.
		exceeded, err := isStandingApprovalLimitReached(ctx, tx, profile.ID)
		if err != nil {
			return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("internal error")
		}
		if exceeded {
			return &applyOneResult{errorCode: string(ErrStandingApprovalLimitReached)}, fmt.Errorf("standing approval limit reached")
		}

		var expiresAt *time.Time
		startsAt := time.Now().UTC()
		if spec.DurationDays != nil {
			if *spec.DurationDays <= 0 {
				return &applyOneResult{errorCode: string(ErrInvalidRequest)}, fmt.Errorf("template standing_approval has invalid duration_days")
			}
			t := startsAt.Add(time.Duration(*spec.DurationDays) * 24 * time.Hour)
			expiresAt = &t
		}

		var maxExec *int
		if spec.MaxExecutions != nil {
			if *spec.MaxExecutions < 1 {
				return &applyOneResult{errorCode: string(ErrInvalidRequest)}, fmt.Errorf("template standing_approval has invalid max_executions")
			}
			maxExec = spec.MaxExecutions
		}

		saID, err := generatePrefixedID("sa_", 16)
		if err != nil {
			return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("internal error")
		}

		srcID := ac.ID
		sa, err := db.CreateStandingApproval(ctx, tx, db.CreateStandingApprovalParams{
			StandingApprovalID:          saID,
			AgentID:                     agentID,
			UserID:                      profile.ID,
			ActionType:                  actionType,
			ActionVersion:               "1",
			Constraints:                 standingBytes,
			SourceActionConfigurationID: &srcID,
			MaxExecutions:               maxExec,
			StartsAt:                    startsAt,
			ExpiresAt:                   expiresAt,
		})
		if err != nil {
			var saErr *db.StandingApprovalError
			if errors.As(err, &saErr) && saErr.Code == db.StandingApprovalErrAgentNotFound {
				return &applyOneResult{errorCode: string(ErrAgentNotFound)}, fmt.Errorf("agent not found")
			}
			return &applyOneResult{errorCode: string(ErrInternalError)}, fmt.Errorf("internal error")
		}
		saOut = sa
	}

	var linkedSA []db.StandingApproval
	if saOut != nil {
		linkedSA = append(linkedSA, *saOut)
	}
	acResp := toActionConfigResponseWithLinked(*ac, linkedSA)
	result := &applyOneResult{actionConfig: &acResp}
	if saOut != nil {
		s := toStandingApprovalResponse(*saOut)
		result.standingApproval = &s
	}

	return result, nil
}

// isStandingApprovalLimitReached checks whether the user has reached their
// plan's standing approval limit. Returns (exceeded, error).
func isStandingApprovalLimitReached(ctx context.Context, d db.DBTX, userID string) (bool, error) {
	sp, err := db.GetSubscriptionWithPlan(ctx, d, userID)
	if err != nil {
		return false, fmt.Errorf("get subscription: %w", err)
	}
	if sp == nil {
		return false, nil // no subscription — bypass limits
	}
	eff := sp.EffectiveQuotaPlan()
	limit := eff.MaxStandingApprovals
	if limit == nil {
		return false, nil // unlimited
	}
	count, err := db.CountActiveStandingApprovalsByUser(ctx, d, userID)
	if err != nil {
		return false, fmt.Errorf("count standing approvals: %w", err)
	}
	return count >= *limit, nil
}

// deduplicateStrings returns a new slice with duplicates removed, preserving
// the first occurrence order.
func deduplicateStrings(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
