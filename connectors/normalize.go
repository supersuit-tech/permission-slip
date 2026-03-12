package connectors

import "encoding/json"

// NormalizeParameters rewrites alias keys in params to their canonical names
// according to the provided alias map. Returns the original bytes unchanged if
// no aliases match, the map is empty, or params cannot be parsed (fail-open).
//
// If both an alias key and the canonical key are present, the canonical value
// is kept and the alias key is dropped.
//
// Alias chains are not supported: each alias must map directly to a final
// canonical key. If "a"→"b" and "b"→"c" are both in the map, the result is
// undefined — the iteration order of Go maps is non-deterministic.
func NormalizeParameters(aliases map[string]string, params json.RawMessage) json.RawMessage {
	if len(aliases) == 0 || len(params) == 0 {
		return params
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(params, &m); err != nil {
		return params
	}

	changed := false
	for alias, canonical := range aliases {
		if _, hasCanonical := m[canonical]; hasCanonical {
			// Canonical key already present — drop the alias if also present.
			if _, hasAlias := m[alias]; hasAlias {
				delete(m, alias)
				changed = true
			}
			continue
		}
		if val, hasAlias := m[alias]; hasAlias {
			m[canonical] = val
			delete(m, alias)
			changed = true
		}
	}

	if !changed {
		return params
	}

	out, err := json.Marshal(m)
	if err != nil {
		return params
	}
	return out
}
