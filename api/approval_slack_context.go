package api

import (
	"encoding/json"
)

// mergeSlackContextFromResourceDetailsIntoContext copies slack_context from resolved
// resource_details into context.details.slack_context for approval UI (issue #981).
// If resource_details is nil or has no slack_context, returns contextJSON unchanged.
func mergeSlackContextFromResourceDetailsIntoContext(contextJSON json.RawMessage, resourceDetails []byte) json.RawMessage {
	if len(resourceDetails) == 0 {
		return contextJSON
	}
	var rd map[string]json.RawMessage
	if err := json.Unmarshal(resourceDetails, &rd); err != nil {
		return contextJSON
	}
	scRaw, ok := rd["slack_context"]
	if !ok || len(scRaw) == 0 {
		return contextJSON
	}

	var ctxObj map[string]json.RawMessage
	if err := json.Unmarshal(contextJSON, &ctxObj); err != nil || ctxObj == nil {
		return contextJSON
	}
	detailsRaw, hasDetails := ctxObj["details"]
	var detailsObj map[string]json.RawMessage
	if hasDetails && len(detailsRaw) > 0 {
		_ = json.Unmarshal(detailsRaw, &detailsObj)
	}
	if detailsObj == nil {
		detailsObj = map[string]json.RawMessage{}
	}
	if _, exists := detailsObj["slack_context"]; exists {
		return contextJSON
	}
	detailsObj["slack_context"] = scRaw
	mergedDetails, err := json.Marshal(detailsObj)
	if err != nil {
		return contextJSON
	}
	ctxObj["details"] = mergedDetails
	out, err := json.Marshal(ctxObj)
	if err != nil {
		return contextJSON
	}
	if len(out) > maxApprovalContextJSONBytes {
		return contextJSON
	}
	return out
}
