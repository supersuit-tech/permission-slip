package connectors_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"

	// Blank-import to trigger init() self-registration.
	_ "github.com/supersuit-tech/permission-slip/connectors/all"
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
//   - GitHub: connectors/github/resolve_resource_details.go
//     • resolveWorkflow → workflow_name
//     • resolveWebhook  → webhook_url, webhook_events
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
	// GitHub (see connectors/github/resolve_resource_details.go)
	"workflow_name":  true,
	"webhook_url":    true,
	"webhook_events": true,
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

// TestAllActionsHaveDisplayTemplate ensures that once a connector starts
// defining DisplayTemplates, ALL of its actions have one. This prevents
// partial adoption where some actions render nicely and others fall back to
// the confusing generic summary (#862).
//
// Connectors that haven't adopted templates yet are skipped — this allows
// a graduated rollout without blocking unrelated PRs.
func TestAllActionsHaveDisplayTemplate(t *testing.T) {
	for _, c := range connectors.BuiltInConnectors() {
		mp, ok := c.(connectors.ManifestProvider)
		if !ok {
			continue
		}
		manifest := mp.Manifest()

		// Check whether this connector has adopted templates at all.
		hasAny := false
		for _, action := range manifest.Actions {
			if action.DisplayTemplate != "" {
				hasAny = true
				break
			}
		}
		if !hasAny {
			continue // connector hasn't started template adoption yet
		}

		for _, action := range manifest.Actions {
			if action.DisplayTemplate == "" {
				t.Errorf("%s: action %q is missing a DisplayTemplate — once a connector defines any templates, all its actions must have one",
					manifest.ID, action.ActionType)
			}
		}
	}
}
