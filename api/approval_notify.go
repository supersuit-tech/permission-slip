package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/notify"
)

// NotifyApprovalRequest dispatches a notification to the approver for a newly
// created one-off approval request. Call this after successfully inserting the
// approval row. It is fire-and-forget: errors are logged, not returned, so
// the HTTP response is not blocked by delivery latency.
//
// The future POST /v1/approvals/request handler should call this after
// creating the approval row:
//
//	NotifyApprovalRequest(r.Context(), deps, &approval, agent, approverProfile)
func NotifyApprovalRequest(ctx context.Context, deps *Deps, approval *db.Approval, agent *db.Agent, approver *db.Profile) {
	if deps.Notifier == nil {
		return
	}

	if deps.BaseURL == "" {
		log.Printf("notify: skipping notification for %s — BASE_URL is not configured", approval.ApprovalID)
		return
	}

	agentName := extractAgentName(agent)
	approvalURL := fmt.Sprintf("%s/approve/%s", deps.BaseURL, approval.ApprovalID)


	notifApproval := notify.Approval{
		ApprovalID:  approval.ApprovalID,
		AgentID:     approval.AgentID,
		AgentName:   agentName,
		Action:      json.RawMessage(approval.Action),
		Context:     json.RawMessage(approval.Context),
		ApprovalURL: approvalURL,
		ExpiresAt:   approval.ExpiresAt,
		CreatedAt:   approval.CreatedAt,
	}

	recipient := notify.Recipient{
		UserID:   approver.ID,
		Username: approver.Username,
		Email:    approver.Email,
		Phone:    approver.Phone,
	}

	deps.Notifier.Dispatch(ctx, notifApproval, recipient)
	log.Printf("notify: dispatched approval notification for %s to user %s", approval.ApprovalID, approver.ID)
}

// extractAgentName pulls the human-readable name from agent metadata JSONB.
// Falls back to "Agent <id>" if no name is set.
func extractAgentName(agent *db.Agent) string {
	if len(agent.Metadata) > 0 {
		var meta map[string]json.RawMessage
		if json.Unmarshal(agent.Metadata, &meta) == nil {
			if raw, ok := meta["name"]; ok {
				var name string
				if json.Unmarshal(raw, &name) == nil && name != "" {
					return name
				}
			}
		}
	}
	return fmt.Sprintf("Agent %d", agent.AgentID)
}
