package api

import (
	"encoding/json"
)

// standingApprovalAutoApproveOutcome reports the result of the standing-approval
// auto-approve probe before falling through to a pending approval.
type standingApprovalAutoApproveOutcome struct {
	Handled           bool
	FallthroughReason string
}

const standingApprovalFallthroughMetadataUnavailable = "metadata_unavailable"

// mergeStandingApprovalFallthroughIntoContext copies standing_approval_fallthrough
// into context.details when auto-approve was skipped for a known reason.
func mergeStandingApprovalFallthroughIntoContext(contextJSON json.RawMessage, reason string) json.RawMessage {
	if reason == "" {
		return contextJSON
	}

	var message string
	switch reason {
	case standingApprovalFallthroughMetadataUnavailable:
		message = "A standing approval rule may apply, but verified email metadata was unavailable so it could not be evaluated automatically."
	default:
		return contextJSON
	}

	var ctxObj map[string]json.RawMessage
	if len(contextJSON) > 0 {
		if err := json.Unmarshal(contextJSON, &ctxObj); err != nil {
			return contextJSON
		}
	} else {
		ctxObj = map[string]json.RawMessage{}
	}

	var details map[string]any
	if raw, ok := ctxObj["details"]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &details)
	}
	if details == nil {
		details = map[string]any{}
	}
	details["standing_approval_fallthrough"] = map[string]string{
		"reason":  reason,
		"message": message,
	}

	encoded, err := json.Marshal(details)
	if err != nil {
		return contextJSON
	}
	ctxObj["details"] = encoded

	out, err := json.Marshal(ctxObj)
	if err != nil {
		return contextJSON
	}
	return out
}
