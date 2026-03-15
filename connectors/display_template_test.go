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

// TestDisplayTemplateParamsExist validates that every {{param}} reference in a
// display_template actually exists in the action's ParametersSchema properties.
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
				if _, exists := schema.Properties[paramName]; !exists {
					t.Errorf("%s: display_template references {{%s}} but parameter %q is not in parameters_schema properties",
						action.ActionType, match[0], paramName)
				}
			}
		}
	}
}
