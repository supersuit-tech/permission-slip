package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/notify"
)

// NotifyStandingApprovalRequest dispatches email/push for a new rule proposal.
func NotifyStandingApprovalRequest(ctx context.Context, deps *Deps, sar *db.StandingApprovalRequest, agent *db.Agent, approver *db.Profile) {
	if deps.Notifier == nil {
		return
	}
	if deps.BaseURL == "" {
		log.Printf("notify: skipping standing approval request notification for %s — BASE_URL is not configured", sar.RequestID)
		return
	}

	agentName := extractAgentName(agent)
	approvalURL := fmt.Sprintf("%s/approve-rule/%s", deps.BaseURL, sar.RequestID)

	actionPayload, err := json.Marshal(map[string]any{
		"type":                       sar.ActionType,
		"constraints":                json.RawMessage(sar.Constraints),
		"connector_name":             sar.ConnectorName,
		"connector_instance_display": sar.ConnectorInstanceDisplay,
	})
	if err != nil {
		log.Printf("notify: marshal standing approval request action: %v", err)
		return
	}

	ctxPayload, err := json.Marshal(map[string]any{
		"description": standingApprovalRequestDescription(sar),
		"kind":        "standing_approval_request",
	})
	if err != nil {
		log.Printf("notify: marshal standing approval request context: %v", err)
		return
	}

	// Display-only expiry for notification templates (proposal review window, not rule lifetime).
	expiresAt := sar.CreatedAt.Add(30 * 24 * time.Hour)

	notifApproval := notify.Approval{
		ApprovalID:  sar.RequestID,
		AgentID:     sar.AgentID,
		AgentName:   agentName,
		Action:      actionPayload,
		Context:     ctxPayload,
		ApprovalURL: approvalURL,
		ExpiresAt:   expiresAt,
		CreatedAt:   sar.CreatedAt,
		Type:        notify.NotificationTypeStandingApprovalRequest,
	}

	recipient := notify.Recipient{
		UserID:   approver.ID,
		Username: approver.Username,
		Email:    approver.Email,
		Phone:    approver.Phone,
	}

	deps.Notifier.Dispatch(ctx, notifApproval, recipient)
	log.Printf("notify: dispatched standing approval request notification for %s to user %s", sar.RequestID, approver.ID)
}

func standingApprovalRequestDescription(sar *db.StandingApprovalRequest) string {
	label := sar.ActionType
	if sar.ConnectorName != nil && *sar.ConnectorName != "" {
		label = *sar.ConnectorName
		if sar.ConnectorInstanceDisplay != nil && *sar.ConnectorInstanceDisplay != "" {
			label = fmt.Sprintf("%s (%s)", *sar.ConnectorName, *sar.ConnectorInstanceDisplay)
		}
	}
	return fmt.Sprintf("Proposed auto-approve rule for %s", label)
}
