package connectors_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"

	// Blank-import to trigger init() self-registration.
	_ "github.com/supersuit-tech/permission-slip-web/connectors/all"
)

// templateParamPattern matches {{param}} and {{param:directive}} placeholders.
var templateParamPattern = regexp.MustCompile(`\{\{(\w+)(?::\w+)?\}\}`)

// resourceDetailFields are keys that come from ResolveResourceDetails at
// runtime, not from the action's ParametersSchema. Templates can reference
// these because the frontend merges resourceDetails into the template lookup
// (see ActionPreviewSummary.tsx buildParts and mobile approvalUtils.ts).
//
// When adding a new connector that implements ResourceDetailResolver, add its
// returned keys here. Source files for each connector's resolver:
//   - Slack: connectors/slack/resolve_resource_details.go
//     • resolveChannel → channel_name
//     • resolveUser    → user_name
//   - Google: connectors/google/resolve_resource_details.go
//     • resolveCalendarEvent → title, start_time
//     • resolveFile          → file_name, title
//     • resolveEmail         → subject, from
//     • resolveSheet         → title, range
var resourceDetailFields = map[string]bool{
	// Slack (see connectors/slack/resolve_resource_details.go)
	"channel_name": true,
	"user_name":    true,
	// Google (see connectors/google/resolve_resource_details.go)
	"title":      true,
	"file_name":  true,
	"start_time": true,
	"subject":    true,
	"from":       true,
	"range":      true,
}

// TestDisplayTemplateParamsExist validates that every {{param}} reference in a
// display_template actually exists in the action's ParametersSchema properties
// or is a known resource_details field populated at runtime.
// This catches typos like {{start_tme:datetime}} at test time rather than
// silently falling through to raw values at runtime.
func TestDisplayTemplateParamsExist(t *testing.T) {
	for _, c := range connectors.BuiltInConnectors() {
		mp, ok := c.(connectors.ManifestProvider)
		if !ok {
			continue
		}
		manifest := mp.Manifest()
		for _, action := range manifest.Actions {
			if action.DisplayTemplate == "" {
				continue
			}

			// Parse the schema to extract property names.
			var schema struct {
				Properties map[string]json.RawMessage `json:"properties"`
			}
			if len(action.ParametersSchema) > 0 {
				if err := json.Unmarshal(action.ParametersSchema, &schema); err != nil {
					t.Errorf("%s: failed to parse parameters_schema: %v", action.ActionType, err)
					continue
				}
			}

			// Extract all {{param}} references from the template.
			matches := templateParamPattern.FindAllStringSubmatch(action.DisplayTemplate, -1)
			for _, match := range matches {
				paramName := match[1]
				if _, exists := schema.Properties[paramName]; exists {
					continue
				}
				if resourceDetailFields[paramName] {
					continue
				}
				t.Errorf("%s: display_template references {{%s}} but parameter %q is not in parameters_schema properties or known resource_details fields",
					action.ActionType, match[0], paramName)
			}
		}
	}
}

// TestAllActionsHaveDisplayTemplate ensures every connector action defines a
// DisplayTemplate. Templates are the primary way human-readable summaries are
// rendered in approval cards. Without one, the UI falls back to a generic
// summary that may be confusing (see #862).
func TestAllActionsHaveDisplayTemplate(t *testing.T) {
	for _, c := range connectors.BuiltInConnectors() {
		mp, ok := c.(connectors.ManifestProvider)
		if !ok {
			continue
		}
		manifest := mp.Manifest()
		for _, action := range manifest.Actions {
			if action.DisplayTemplate == "" {
				t.Errorf("%s: action %q is missing a DisplayTemplate — every action must have one for readable approval summaries",
					manifest.ID, action.ActionType)
			}
		}
	}
}
