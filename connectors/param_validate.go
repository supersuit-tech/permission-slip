package connectors

import "encoding/json"

// ParamValidatorFunc is a function that validates raw JSON parameters for a
// single action type. Connectors register one per action in their
// ValidateParams dispatch table.
type ParamValidatorFunc func(params json.RawMessage) error

// ValidateWithMap is a helper for implementing ParamValidator.ValidateParams.
// It looks up the action type in the provided map and calls the corresponding
// validator. Returns nil (fail-open) if the action type has no entry.
//
// Usage in a connector:
//
//	func (c *FooConnector) ValidateParams(actionType string, params json.RawMessage) error {
//	    return connectors.ValidateWithMap(actionType, params, c.validators())
//	}
func ValidateWithMap(actionType string, params json.RawMessage, validators map[string]ParamValidatorFunc) error {
	fn, ok := validators[actionType]
	if !ok {
		return nil // fail-open: no validator registered for this action
	}
	return fn(params)
}
