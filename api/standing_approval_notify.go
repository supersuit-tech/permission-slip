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

// NotifyStandingApprovalExecution dispatches an informational notification to
// the user when an agent executes an action via a standing approval. This is
// fire-and-forget: errors are logged, not returned, so the HTTP response is
// never blocked by delivery latency.
//
// Call this after a successful RecordStandingApprovalExecutionByAgent +
// connector execution in the standing approval path.
func NotifyStandingApprovalExecution(ctx context.Context, deps *Deps, exec *db.StandingApprovalExecution, agent *db.Agent, actionType string, parameters json.RawMessage) {
	if deps.Notifier == nil {
		return
	}

	if deps.BaseURL == "" {
		log.Printf("notify: skipping standing execution notification — BASE_URL is not configured")
		return
	}

	// Fetch the user's profile for recipient contact info.
	profile, err := db.GetProfileByUserID(ctx, deps.DB, exec.UserID)
	if err != nil {
		log.Printf("notify: standing execution: failed to fetch profile for user %s: %v", exec.UserID, err)
		CaptureError(ctx, err)
		return
	}

	agentName := extractAgentName(agent)
	activityURL := fmt.Sprintf("%s/activity", deps.BaseURL)

	// Standing execution templates do not use Context; omit empty object noise.
	var contextJSON json.RawMessage

	// Build the action JSON with type and parameters for the notification
	// templates (used for parameter summary in emails).
	actionJSON := buildActionJSON(actionType, parameters)

	approval := notify.Approval{
		ApprovalID:  exec.StandingApprovalID,
		AgentID:     exec.AgentID,
		AgentName:   agentName,
		Action:      actionJSON,
		Context:     contextJSON,
		ApprovalURL: activityURL,
		CreatedAt:   time.Now(),
		Type:        notify.NotificationTypeStandingExecution,
	}

	enabled, err := db.IsNotificationTypeEnabled(ctx, deps.DB, exec.UserID, db.NotificationTypeStandingExecution)
	if err != nil {
		log.Printf("notify: standing execution: failed to load notification type preference for user %s: %v", exec.UserID, err)
		CaptureError(ctx, err)
		return
	}
	if !enabled {
		log.Printf("notify: skipping standing execution notification for user %s — notification type disabled", exec.UserID)
		return
	}

	recipient := notify.Recipient{
		UserID:   profile.ID,
		Username: profile.Username,
		Email:    profile.Email,
		Phone:    profile.Phone,
	}

	deps.Notifier.Dispatch(ctx, approval, recipient)
	log.Printf("notify: dispatched standing execution notification for SA %s to user %s", exec.StandingApprovalID, exec.UserID)
}

// buildActionJSON constructs the action JSON expected by notification templates:
// {"type": "...", "parameters": {...}}
func buildActionJSON(actionType string, parameters json.RawMessage) json.RawMessage {
	action := map[string]json.RawMessage{
		"type": mustMarshal(actionType),
	}
	if len(parameters) > 0 {
		action["parameters"] = parameters
	}
	result, _ := json.Marshal(action)
	return result
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
