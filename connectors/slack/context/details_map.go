package context

import (
	"encoding/json"
)

// DetailsResourceMap returns resource_details entries for Slack approval context.
// Keys match context.details.slack_context in the OpenAPI contract (issue #981).
func DetailsResourceMap(sc *SlackContext) map[string]any {
	if sc == nil {
		return nil
	}
	b, err := json.Marshal(sc)
	if err != nil {
		return nil
	}
	var asMap map[string]any
	if err := json.Unmarshal(b, &asMap); err != nil {
		return nil
	}
	return map[string]any{"slack_context": asMap}
}
