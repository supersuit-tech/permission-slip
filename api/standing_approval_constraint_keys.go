package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

// validateStandingApprovalConstraintsForAction validates and normalizes standing
// approval constraints, then checks parameter keys against the action schema and
// $meta keys against connector metadata capabilities.
func validateStandingApprovalConstraintsForAction(
	ctx context.Context,
	d db.DBTX,
	registry *connectors.Registry,
	actionType string,
	raw json.RawMessage,
) ([]byte, error) {
	normalized, err := validateStandingApprovalConstraints(raw)
	if err != nil {
		return nil, err
	}
	if err := validateStandingApprovalConstraintKeys(ctx, d, registry, actionType, normalized); err != nil {
		return nil, err
	}
	return fillMissingSchemaParameterWildcards(ctx, d, actionType, normalized)
}

// fillMissingSchemaParameterWildcards adds "*" for every action-schema parameter
// key absent from the constraints object. Unset fields mean "any value" for
// standing approvals — without this, $meta-only rules reject execution params
// as extra keys before metadata matching runs.
func fillMissingSchemaParameterWildcards(
	ctx context.Context,
	d db.DBTX,
	actionType string,
	constraints []byte,
) ([]byte, error) {
	schemaKeys, err := actionSchemaPropertyKeys(ctx, d, actionType)
	if err != nil {
		return nil, err
	}
	if len(schemaKeys) == 0 {
		return constraints, nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(constraints, &obj); err != nil {
		return nil, fmt.Errorf("constraints must be a JSON object")
	}

	mutated := false
	for key := range schemaKeys {
		if _, ok := obj[key]; !ok {
			obj[key] = json.RawMessage(`"*"`)
			mutated = true
		}
	}

	if !mutated {
		return constraints, nil
	}

	return json.Marshal(obj)
}

func validateStandingApprovalConstraintKeys(
	ctx context.Context,
	d db.DBTX,
	registry *connectors.Registry,
	actionType string,
	constraints []byte,
) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(constraints, &obj); err != nil {
		return fmt.Errorf("constraints must be a JSON object")
	}

	schemaKeys, err := actionSchemaPropertyKeys(ctx, d, actionType)
	if err != nil {
		return err
	}

	var metaFields map[string]struct{}
	var metaSupported bool
	if registry != nil {
		metaFields, metaSupported = connectorMetaConstraintFields(registry, actionType)
	}

	metaRaw, hasMeta := obj[db.MetaNamespaceKey]
	if hasMeta {
		var metaObj map[string]json.RawMessage
		if err := json.Unmarshal(metaRaw, &metaObj); err != nil {
			return fmt.Errorf("$meta constraints must be a JSON object")
		}
		if len(metaObj) > 0 {
			if !metaSupported {
				return fmt.Errorf("$meta constraints are not supported for action %q", actionType)
			}
			for key := range metaObj {
				if _, ok := metaFields[key]; !ok {
					return fmt.Errorf("unsupported $meta constraint key %q for action %q", key, actionType)
				}
			}
		}
	}

	dataWindowRaw, hasDataWindow := obj[db.DataWindowNamespaceKey]
	if hasDataWindow {
		if err := db.ValidateDataWindowConstraintShape(dataWindowRaw); err != nil {
			return err
		}
		pair, err := db.GetActionDataWindowParams(ctx, d, actionType)
		if err != nil {
			return fmt.Errorf("lookup action data window: %w", err)
		}
		if pair == nil {
			return fmt.Errorf("$data_window constraints are not supported for action %q", actionType)
		}
	}

	for key, val := range obj {
		if key == db.MetaNamespaceKey || key == db.DataWindowNamespaceKey {
			continue
		}
		if key == "" {
			return formatEmptyConstraintKeyError(obj[key], actionType, metaFields)
		}
		if schemaKeys != nil {
			if _, ok := schemaKeys[key]; !ok {
				return formatUnknownConstraintKeyError(key, actionType, metaFields)
			}
			continue
		}
		if metaFields != nil {
			if _, ok := metaFields[key]; ok {
				return formatUnknownConstraintKeyError(key, actionType, metaFields)
			}
		}
		if isAllowedParameterConstraintValue(val) {
			continue
		}
		return formatUnknownConstraintKeyError(key, actionType, metaFields)
	}

	return nil
}

func isAllowedParameterConstraintValue(val json.RawMessage) bool {
	var s string
	if json.Unmarshal(val, &s) == nil {
		return true
	}
	var patternOnly map[string]json.RawMessage
	if err := json.Unmarshal(val, &patternOnly); err != nil {
		return false
	}
	if len(patternOnly) != 1 {
		return false
	}
	_, ok := patternOnly[db.PatternKey]
	return ok
}

func actionSchemaPropertyKeys(ctx context.Context, d db.DBTX, actionType string) (map[string]struct{}, error) {
	schema, err := db.GetActionParametersSchema(ctx, d, actionType)
	if err != nil {
		return nil, fmt.Errorf("lookup action schema: %w", err)
	}
	if schema == nil || len(schema.Schema) == 0 {
		return nil, nil
	}
	keys, err := extractSchemaPropertyKeys(schema.Schema)
	if err != nil {
		return nil, fmt.Errorf("parse action schema: %w", err)
	}
	return keys, nil
}

func extractSchemaPropertyKeys(schemaJSON []byte) (map[string]struct{}, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(schemaJSON, &root); err != nil {
		return nil, err
	}

	keys := make(map[string]struct{})
	if props := root["properties"]; len(props) > 0 {
		if err := mergePropertyKeys(props, keys); err != nil {
			return nil, err
		}
	}
	if anyOf := root["anyOf"]; len(anyOf) > 0 {
		var branches []map[string]json.RawMessage
		if err := json.Unmarshal(anyOf, &branches); err != nil {
			return nil, err
		}
		for _, branch := range branches {
			if props := branch["properties"]; len(props) > 0 {
				if err := mergePropertyKeys(props, keys); err != nil {
					return nil, err
				}
			}
		}
	}
	return keys, nil
}

func mergePropertyKeys(propsJSON json.RawMessage, keys map[string]struct{}) error {
	var props map[string]json.RawMessage
	if err := json.Unmarshal(propsJSON, &props); err != nil {
		return err
	}
	for name := range props {
		keys[name] = struct{}{}
	}
	return nil
}

func connectorMetaConstraintFields(registry *connectors.Registry, actionType string) (map[string]struct{}, bool) {
	fields, ok := actionMetaConstraintFieldList(registry, actionType)
	if !ok {
		return nil, false
	}
	out := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		out[field] = struct{}{}
	}
	return out, true
}

func actionMetaConstraintFieldList(registry *connectors.Registry, actionType string) ([]string, bool) {
	if registry == nil {
		return nil, false
	}
	parts := strings.SplitN(actionType, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	conn, ok := registry.Get(parts[0])
	if !ok {
		return nil, false
	}
	capabilities, ok := conn.(connectors.ConstraintMetadataCapabilities)
	if !ok {
		return nil, false
	}
	fields, supported := capabilities.ConstraintMetadataActionSupport(actionType)
	if !supported || len(fields) == 0 {
		return nil, false
	}
	out := append([]string(nil), fields...)
	sort.Strings(out)
	return out, true
}

func formatEmptyConstraintKeyError(value json.RawMessage, actionType string, metaFields map[string]struct{}) error {
	var innerObj map[string]json.RawMessage
	if err := json.Unmarshal(value, &innerObj); err == nil {
		for innerKey := range innerObj {
			if metaFields != nil {
				if _, ok := metaFields[innerKey]; ok {
					return fmt.Errorf(
						`constraint keys must not be empty; did you mean a verified metadata constraint? Use {"$meta":{%q:...}}`,
						innerKey,
					)
				}
			}
		}
	}
	return fmt.Errorf("constraint keys must not be empty")
}

func formatUnknownConstraintKeyError(key, actionType string, metaFields map[string]struct{}) error {
	if metaFields != nil {
		if _, ok := metaFields[key]; ok {
			return fmt.Errorf(
				`constraint key %q is not a parameter on action %q; did you mean a verified metadata constraint? Use {"$meta":{%q:...}}`,
				key, actionType, key,
			)
		}
	}
	return fmt.Errorf("constraint key %q is not a parameter on action %q", key, actionType)
}
