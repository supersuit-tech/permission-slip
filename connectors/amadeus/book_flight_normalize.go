package amadeus

import "encoding/json"

// snakeToCamelTravelerFields maps snake_case top-level traveler field names
// to their camelCase equivalents expected by the Amadeus API.
var snakeToCamelTravelerFields = map[string]string{
	"date_of_birth": "dateOfBirth",
}

// snakeToCamelNameFields maps snake_case name sub-object field names
// to their camelCase equivalents expected by the Amadeus API.
var snakeToCamelNameFields = map[string]string{
	"first_name": "firstName",
	"last_name":  "lastName",
}

// Normalize rewrites snake_case traveler fields to camelCase inside the
// travelers array. This allows agents to follow the system-wide snake_case
// convention while the stored/executed parameters match the Amadeus API.
//
// Fields that are already camelCase pass through unchanged.
func (a *bookFlightAction) Normalize(params json.RawMessage) json.RawMessage {
	if len(params) == 0 {
		return params
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(params, &m); err != nil {
		return params
	}

	travelersRaw, ok := m["travelers"]
	if !ok {
		return params
	}

	var travelers []json.RawMessage
	if err := json.Unmarshal(travelersRaw, &travelers); err != nil {
		return params
	}

	changed := false
	for i, raw := range travelers {
		normalized, didChange := normalizeTraveler(raw)
		if didChange {
			travelers[i] = normalized
			changed = true
		}
	}

	if !changed {
		return params
	}

	updated, err := json.Marshal(travelers)
	if err != nil {
		return params
	}
	m["travelers"] = updated

	result, err := json.Marshal(m)
	if err != nil {
		return params
	}
	return result
}

// normalizeTraveler rewrites snake_case keys in a single traveler object:
//   - top-level: date_of_birth → dateOfBirth
//   - nested name: first_name → firstName, last_name → lastName
func normalizeTraveler(raw json.RawMessage) (json.RawMessage, bool) {
	var t map[string]json.RawMessage
	if err := json.Unmarshal(raw, &t); err != nil {
		return raw, false
	}

	changed := false

	// Rewrite top-level snake_case fields (e.g., date_of_birth → dateOfBirth).
	// Name fields (first_name, last_name) are NOT normalized at the top level —
	// they belong inside the name sub-object per the Amadeus API schema.
	for snake, camel := range snakeToCamelTravelerFields {
		if _, hasCamel := t[camel]; hasCamel {
			if _, hasSnake := t[snake]; hasSnake {
				delete(t, snake)
				changed = true
			}
			continue
		}
		if val, hasSnake := t[snake]; hasSnake {
			t[camel] = val
			delete(t, snake)
			changed = true
		}
	}

	// Rewrite name sub-object fields (first_name → firstName, last_name → lastName).
	if nameRaw, ok := t["name"]; ok {
		var name map[string]json.RawMessage
		if err := json.Unmarshal(nameRaw, &name); err == nil {
			nameChanged := false
			for snake, camel := range snakeToCamelNameFields {
				if _, hasCamel := name[camel]; hasCamel {
					if _, hasSnake := name[snake]; hasSnake {
						delete(name, snake)
						nameChanged = true
					}
					continue
				}
				if val, hasSnake := name[snake]; hasSnake {
					name[camel] = val
					delete(name, snake)
					nameChanged = true
				}
			}
			if nameChanged {
				if updated, err := json.Marshal(name); err == nil {
					t["name"] = updated
					changed = true
				}
			}
		}
	}

	if !changed {
		return raw, false
	}

	result, err := json.Marshal(t)
	if err != nil {
		return raw, false
	}
	return result, true
}
