package api

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// mergeEmailThreadFromResourceDetailsIntoContext copies email_thread from resolved
// resource_details into context.details.email_thread for approval UI (issue #975).
// If resource_details is nil or has no email_thread, returns contextJSON unchanged.
func mergeEmailThreadFromResourceDetailsIntoContext(contextJSON json.RawMessage, resourceDetails []byte) json.RawMessage {
	if len(resourceDetails) == 0 {
		return contextJSON
	}
	var rd map[string]json.RawMessage
	if err := json.Unmarshal(resourceDetails, &rd); err != nil {
		return contextJSON
	}
	threadRaw, ok := rd["email_thread"]
	if !ok || len(threadRaw) == 0 {
		return contextJSON
	}
	var thread connectors.EmailThreadPayload
	if err := json.Unmarshal(threadRaw, &thread); err != nil {
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
	if _, exists := detailsObj["email_thread"]; exists {
		return contextJSON
	}
	detailsObj["email_thread"] = threadRaw
	mergedDetails, err := json.Marshal(detailsObj)
	if err != nil {
		return contextJSON
	}
	ctxObj["details"] = mergedDetails
	out, err := json.Marshal(ctxObj)
	if err != nil {
		return contextJSON
	}
	// Preserve the 65536-byte cap from the request validator.
	if len(out) > 65536 {
		return contextJSON
	}
	return out
}
