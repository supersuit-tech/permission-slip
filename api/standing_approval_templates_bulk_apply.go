package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip/db"
)

type bulkApplyStandingApprovalTemplateRequest struct {
	AgentID     int64    `json:"agent_id" validate:"gt=0"`
	TemplateIDs []string `json:"template_ids" validate:"required,min=1,max=50,dive,required,max=255"`
}

type bulkApplyResultError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type bulkApplyStandingApprovalResult struct {
	TemplateID       string                    `json:"template_id"`
	Success          bool                      `json:"success"`
	StandingApproval *standingApprovalResponse `json:"standing_approval,omitempty"`
	Error            *bulkApplyResultError     `json:"error,omitempty"`
}

type bulkApplyStandingApprovalTemplateResponse struct {
	Results []bulkApplyStandingApprovalResult `json:"results"`
}

func handleBulkApplyStandingApprovalTemplates(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req bulkApplyStandingApprovalTemplateRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		uniqueIDs := deduplicateStrings(req.TemplateIDs)

		templates, err := db.GetStandingApprovalTemplatesByIDs(r.Context(), deps.DB, uniqueIDs)
		if err != nil {
			log.Printf("[%s] BulkApplyStandingApprovalTemplates: fetch templates: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
			return
		}

		tplByID := make(map[string]*db.StandingApprovalTemplate, len(templates))
		for i := range templates {
			tplByID[templates[i].ID] = &templates[i]
		}
		for _, id := range uniqueIDs {
			if _, ok := tplByID[id]; !ok {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrStandingApprovalTemplateNotFound,
					fmt.Sprintf("Template %q not found", id)))
				return
			}
		}

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

		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] BulkApplyStandingApprovalTemplates: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx)
		}

		enabled, err := db.AgentConnectorEnabled(r.Context(), tx, req.AgentID, profile.ID, connectorID)
		if err != nil {
			log.Printf("[%s] BulkApplyStandingApprovalTemplates: connector check: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
			return
		}
		if !enabled {
			if _, err := db.EnableAgentConnector(r.Context(), tx, req.AgentID, profile.ID, connectorID); err != nil {
				log.Printf("[%s] BulkApplyStandingApprovalTemplates: enable connector: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
				return
			}
		}

		results := make([]bulkApplyStandingApprovalResult, 0, len(uniqueIDs))
		for _, id := range uniqueIDs {
			tpl := tplByID[id]
			sa, err := applyStandingApprovalTemplateInSavepoint(r.Context(), tx, profile, tpl, req.AgentID)
			if err != nil {
				code := string(ErrInternalError)
				msg := err.Error()
				if httpErr, ok := err.(*applyTemplateHTTPError); ok {
					code = string(httpErr.resp.Error.Code)
					msg = httpErr.resp.Error.Message
				}
				results = append(results, bulkApplyStandingApprovalResult{
					TemplateID: id,
					Success:    false,
					Error:      &bulkApplyResultError{Code: code, Message: msg},
				})
				continue
			}
			s := toStandingApprovalResponse(*sa)
			results = append(results, bulkApplyStandingApprovalResult{
				TemplateID:       id,
				Success:          true,
				StandingApproval: &s,
			})
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] BulkApplyStandingApprovalTemplates: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to apply templates"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, bulkApplyStandingApprovalTemplateResponse{Results: results})
	}
}

func applyStandingApprovalTemplateInSavepoint(
	ctx context.Context,
	tx db.DBTX,
	profile *db.Profile,
	tpl *db.StandingApprovalTemplate,
	agentID int64,
) (*db.StandingApproval, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("internal error")
	}
	sp := "sp_" + hex.EncodeToString(b)

	if _, err := tx.Exec(ctx, "SAVEPOINT "+sp); err != nil {
		return nil, fmt.Errorf("internal error")
	}

	sa, err := applyStandingApprovalTemplateCore(ctx, tx, profile, tpl, agentID)
	if err != nil {
		tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+sp) //nolint:errcheck
		return nil, err
	}

	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT "+sp); err != nil {
		return nil, fmt.Errorf("internal error")
	}
	return sa, nil
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
