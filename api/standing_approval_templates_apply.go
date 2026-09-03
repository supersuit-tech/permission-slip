// Standing-approval templates are disabled for now — endpoints and DB plumbing are
// kept intentionally dormant; see issue #1436.
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

type applyStandingApprovalTemplateRequest struct {
	AgentID int64 `json:"agent_id" validate:"gt=0"`
}

type applyStandingApprovalTemplateResponse struct {
	StandingApproval standingApprovalResponse `json:"standing_approval"`
}

func handleApplyStandingApprovalTemplate(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		templateID := strings.TrimSpace(r.PathValue("id"))
		if templateID == "" || len(templateID) > 255 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "template id is required"))
			return
		}

		var req applyStandingApprovalTemplateRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		tpl, err := db.GetStandingApprovalTemplateByID(r.Context(), deps.DB, templateID)
		if err != nil {
			log.Printf("[%s] ApplyStandingApprovalTemplate: get template: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
			return
		}
		if tpl == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrStandingApprovalTemplateNotFound, "Standing approval template not found"))
			return
		}

		sa, err := applyStandingApprovalTemplateCore(r.Context(), deps.DB, deps.Connectors, profile, tpl, req.AgentID)
		if err != nil {
			if httpErr, ok := err.(*applyTemplateHTTPError); ok {
				RespondError(w, r, httpErr.status, httpErr.resp)
				return
			}
			log.Printf("[%s] ApplyStandingApprovalTemplate: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply template"))
			return
		}

		RespondJSON(w, http.StatusCreated, applyStandingApprovalTemplateResponse{
			StandingApproval: toStandingApprovalResponse(*sa),
		})
	}
}

type applyTemplateHTTPError struct {
	status int
	resp   ErrorResponse
}

func (e *applyTemplateHTTPError) Error() string { return e.resp.Error.Message }

func applyStandingApprovalTemplateCore(
	ctx context.Context,
	pool db.DBTX,
	registry *connectors.Registry,
	profile *db.Profile,
	tpl *db.StandingApprovalTemplate,
	agentID int64,
) (*db.StandingApproval, error) {
	if len(tpl.Name) > shared.ActionConfigNameMaxLength {
		return nil, &applyTemplateHTTPError{http.StatusBadRequest, BadRequest(ErrInvalidRequest, "template name exceeds maximum length")}
	}
	if tpl.Description != nil && len(*tpl.Description) > shared.ActionConfigDescMaxLength {
		return nil, &applyTemplateHTTPError{http.StatusBadRequest, BadRequest(ErrInvalidRequest, "template description exceeds maximum length")}
	}

	constraints := tpl.Constraints
	if len(constraints) == 0 {
		constraints = []byte("{}")
	} else if err := ValidateJSONObject(constraints); err != nil {
		return nil, &applyTemplateHTTPError{http.StatusInternalServerError, InternalError("Invalid template constraints")}
	}

	actionType := strings.TrimSpace(tpl.ActionType)
	if actionType != db.WildcardActionType && strings.Contains(actionType, "*") {
		return nil, &applyTemplateHTTPError{http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid template action_type")}
	}

	standingBytes, err := buildStandingApprovalConstraintsFromTemplate(ctx, pool, registry, actionType, constraints)
	if err != nil {
		return nil, &applyTemplateHTTPError{http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error())}
	}

	tx, owned, err := db.BeginOrContinue(ctx, pool)
	if err != nil {
		return nil, err
	}
	if owned {
		defer db.RollbackTx(ctx, tx)
	}

	enabled, err := db.AgentConnectorEnabled(ctx, tx, agentID, profile.ID, tpl.ConnectorID)
	if err != nil {
		return nil, err
	}
	if !enabled {
		if _, err := db.EnableAgentConnector(ctx, tx, agentID, profile.ID, tpl.ConnectorID); err != nil {
			var acErr *db.AgentConnectorError
			if errors.As(err, &acErr) && acErr.Code == db.AgentConnectorErrAgentNotFound {
				return nil, &applyTemplateHTTPError{http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found")}
			}
			if errors.As(err, &acErr) && acErr.Code == db.AgentConnectorErrConnectorNotFound {
				return nil, &applyTemplateHTTPError{http.StatusBadRequest, BadRequest(ErrInvalidReference, "Invalid connector reference")}
			}
			return nil, err
		}
	}

	var expiresAt *time.Time
	startsAt := time.Now().UTC()
	if tpl.DurationDays != nil {
		if *tpl.DurationDays <= 0 {
			return nil, &applyTemplateHTTPError{http.StatusBadRequest, BadRequest(ErrInvalidRequest, "template has invalid duration_days")}
		}
		t := startsAt.Add(time.Duration(*tpl.DurationDays) * 24 * time.Hour)
		expiresAt = &t
	}

	saID, err := generatePrefixedID("sa_", 16)
	if err != nil {
		return nil, err
	}

	name := tpl.Name
	sa, err := db.CreateStandingApproval(ctx, tx, db.CreateStandingApprovalParams{
		StandingApprovalID: saID,
		AgentID:            agentID,
		UserID:             profile.ID,
		ActionType:         actionType,
		ActionVersion:      "1",
		Constraints:        standingBytes,
		Name:               &name,
		Description:        tpl.Description,
		StartsAt:           startsAt,
		ExpiresAt:          expiresAt,
		Unrestricted:       db.ConstraintsAreUnrestricted(standingBytes),
	})
	if err != nil {
		var saErr *db.StandingApprovalError
		if errors.As(err, &saErr) && saErr.Code == db.StandingApprovalErrAgentNotFound {
			return nil, &applyTemplateHTTPError{http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found")}
		}
		return nil, err
	}

	if owned {
		if err := db.CommitTx(ctx, tx); err != nil {
			return nil, err
		}
	}
	return sa, nil
}

// buildStandingApprovalConstraintsFromTemplate turns template constraint JSON into
// standing-approval constraint JSON.
func buildStandingApprovalConstraintsFromTemplate(
	ctx context.Context,
	pool db.DBTX,
	registry *connectors.Registry,
	actionType string,
	templateConstraints []byte,
) ([]byte, error) {
	validated, err := validateStandingApprovalConstraintsForAction(ctx, pool, registry, actionType, json.RawMessage(templateConstraints))
	if err == nil {
		return validated, nil
	}
	var obj map[string]json.RawMessage
	if jsonErr := json.Unmarshal(templateConstraints, &obj); jsonErr != nil {
		return nil, fmt.Errorf("template constraints must be a JSON object")
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	allBareWildcard := true
	for _, v := range obj {
		var s string
		if json.Unmarshal(v, &s) != nil || s != "*" {
			allBareWildcard = false
			break
		}
	}
	if allBareWildcard {
		return []byte("{}"), nil
	}
	return nil, fmt.Errorf("standing approval constraints could not be derived from template constraints: %w", err)
}
