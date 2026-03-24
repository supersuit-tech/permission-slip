package instacart

import "encoding/json"

// ParameterAliases maps agent-friendly parameter names to the keys Instacart's API expects.
func (a *createProductsLinkAction) ParameterAliases() map[string]string {
	return map[string]string{
		"items": "line_items",
	}
}

// Normalize expands line_items entries that are plain JSON strings into
// {"name": "<string>"} objects so agents can pass
// "line_items": ["milk", "eggs"] while stored parameters match the API shape.
// Runs after flat alias rewriting (e.g. items → line_items) in the API layer.
func (a *createProductsLinkAction) Normalize(params json.RawMessage) json.RawMessage {
	if len(params) == 0 {
		return params
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(params, &m); err != nil {
		return params
	}

	raw, ok := m["line_items"]
	if !ok || len(raw) == 0 {
		return params
	}

	expanded, changed := expandStringLineItemsJSON(raw)
	if !changed {
		return params
	}

	m["line_items"] = expanded
	out, err := json.Marshal(m)
	if err != nil {
		return params
	}
	return out
}

// ValidateRequest rejects parameters that will fail Instacart validation before storage.
func (a *createProductsLinkAction) ValidateRequest(params json.RawMessage) error {
	_, err := parseAndValidateProductsLinkParams(params)
	return err
}

func expandStringLineItemsJSON(raw json.RawMessage) (json.RawMessage, bool) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return raw, false
	}
	out, changed := expandStringLineItemsSlice(items)
	if !changed {
		return raw, false
	}
	updated, err := json.Marshal(out)
	if err != nil {
		return raw, false
	}
	return updated, true
}

func expandStringLineItemsSlice(items []json.RawMessage) ([]json.RawMessage, bool) {
	out := make([]json.RawMessage, len(items))
	changed := false
	for i, raw := range items {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			wrapped, err := json.Marshal(map[string]string{"name": s})
			if err != nil {
				out[i] = raw
				continue
			}
			out[i] = wrapped
			changed = true
			continue
		}
		out[i] = raw
	}
	return out, changed
}

func expandStringLineItemsInPlace(items []json.RawMessage) []json.RawMessage {
	out, changed := expandStringLineItemsSlice(items)
	if !changed {
		return items
	}
	return out
}
