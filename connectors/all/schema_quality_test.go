package all

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// knownMissingAnnotations lists connector fields that are known to be missing
// widget annotations. Each entry is "connector.action_type:field_name".
//
// When you fix one of these, remove it from the list — the test will fail if
// an allowlisted field is actually annotated correctly (i.e., the fix landed
// but the allowlist wasn't updated).
//
// When adding a NEW connector, do NOT add entries here. Fix the annotation
// instead. This list exists only for pre-existing connectors that predate the
// lint.
var knownMissingAnnotations = map[string]bool{
	// textarea — sendgrid HTML content fields
	"sendgrid.sendgrid.send_campaign:html_content":            true,
	"sendgrid.sendgrid.schedule_campaign:html_content":        true,
	"sendgrid.sendgrid.send_transactional_email:html_content": true,

	// datetime — various connectors with ISO 8601 fields missing format
	"asana.asana.create_task:due_at":                         true,
	"asana.asana.update_task:due_at":                         true,
	"dropbox.dropbox.share_file:expires":                     true,
	"hubspot.hubspot.create_deal:closedate":                  true,
	"hubspot.hubspot.update_deal_stage:close_date":           true,
	"jira.jira.create_sprint:start_date":                     true,
	"jira.jira.create_sprint:end_date":                       true,
	"microsoft.microsoft.create_calendar_event:start":        true,
	"microsoft.microsoft.create_calendar_event:end":          true,
	"pagerduty.pagerduty.list_on_call:since":                 true,
	"pagerduty.pagerduty.list_on_call:until":                 true,
	"trello.trello.create_card:due":                          true,
	"zoom.zoom.create_meeting:start_time":                    true,
	"zoom.zoom.update_meeting:start_time":                    true,
}

// annotationIssue describes a single missing annotation found by the lint.
type annotationIssue struct {
	fieldKey   string // "connector.action_type:field_name"
	actionType string
	fieldName  string
	rule       string // "textarea" or "datetime"
	desc       string // original description
	widget     string // current widget (empty = default text)
}

// collectMissingAnnotations scans all built-in connector manifests and returns
// every field that is missing a widget annotation. Both test functions use this
// to avoid duplicating the detection logic.
func collectMissingAnnotations(t *testing.T) []annotationIssue {
	t.Helper()

	var issues []annotationIssue

	for _, c := range connectors.BuiltInConnectors() {
		mp, ok := c.(connectors.ManifestProvider)
		if !ok {
			continue
		}
		m := mp.Manifest()

		for _, action := range m.Actions {
			schema := parseSchema(t, m.ID, action.ActionType, action.ParametersSchema)
			if schema == nil {
				continue
			}

			props, ok := schema["properties"].(map[string]any)
			if !ok {
				continue
			}

			for key, rawProp := range props {
				prop, ok := rawProp.(map[string]any)
				if !ok {
					continue
				}
				propType, _ := prop["type"].(string)
				desc, _ := prop["description"].(string)
				descLower := strings.ToLower(desc)
				widget := extractWidget(prop)
				fieldKey := fmt.Sprintf("%s.%s:%s", m.ID, action.ActionType, key)

				// Rule 1: rich text fields should be textarea
				if propType == "string" && widget != "textarea" {
					if containsAny(descLower, "markdown", "mrkdwn", "html body", "html content") {
						issues = append(issues, annotationIssue{
							fieldKey: fieldKey, actionType: action.ActionType,
							fieldName: key, rule: "textarea", desc: desc, widget: widget,
						})
					}
				}

				// Rule 2: datetime fields should have format: "date-time"
				if propType == "string" && widget != "datetime" {
					format, _ := prop["format"].(string)
					if format != "date-time" &&
						(strings.Contains(descLower, "rfc 3339") || strings.Contains(descLower, "rfc3339") || strings.Contains(descLower, "iso 8601")) &&
						!strings.Contains(descLower, "epoch") {
						issues = append(issues, annotationIssue{
							fieldKey: fieldKey, actionType: action.ActionType,
							fieldName: key, rule: "datetime", desc: desc, widget: widget,
						})
					}
				}
			}
		}
	}

	return issues
}

// TestSchemaWidgetAnnotations is a quality lint that ensures all connector
// schemas have appropriate x-ui widget annotations. Without these, the
// frontend renders default <input type="text"> for fields that should be
// textareas, date pickers, toggles, etc.
//
// This test catches the class of bug fixed in PRs #658 and #659 — where
// markdown body fields rendered as single-line inputs and datetime fields
// rendered as plain text boxes.
func TestSchemaWidgetAnnotations(t *testing.T) {
	t.Parallel()

	for _, issue := range collectMissingAnnotations(t) {
		if knownMissingAnnotations[issue.fieldKey] {
			continue
		}
		switch issue.rule {
		case "textarea":
			t.Errorf("%s field %q: description mentions rich text (%q) but widget is %q — add `\"x-ui\": {\"widget\": \"textarea\"}`",
				issue.actionType, issue.fieldName, issue.desc, widgetOrDefault(issue.widget))
		case "datetime":
			t.Errorf("%s field %q: description mentions datetime format (%q) but missing `\"format\": \"date-time\"` — add it so the frontend renders a date picker",
				issue.actionType, issue.fieldName, issue.desc)
		}
	}
}

// TestKnownMissingAnnotationsAreStillMissing ensures the allowlist stays
// current. If a field is fixed but still in the allowlist, this test fails —
// forcing the developer to remove the stale entry.
func TestKnownMissingAnnotationsAreStillMissing(t *testing.T) {
	t.Parallel()

	actuallyMissing := make(map[string]bool)
	for _, issue := range collectMissingAnnotations(t) {
		actuallyMissing[issue.fieldKey] = true
	}

	for fieldKey := range knownMissingAnnotations {
		if !actuallyMissing[fieldKey] {
			t.Errorf("knownMissingAnnotations contains %q but that field is now properly annotated — remove it from the allowlist", fieldKey)
		}
	}
}

// parseSchema unmarshals a JSON Schema from a RawMessage, returning nil on
// empty or invalid input.
func parseSchema(t *testing.T, connID, actionType string, raw json.RawMessage) map[string]any {
	t.Helper()
	if len(raw) == 0 {
		return nil
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Errorf("%s.%s: invalid parameters_schema JSON: %v", connID, actionType, err)
		return nil
	}
	return schema
}

// extractWidget returns the x-ui.widget value from a property, or "".
func extractWidget(prop map[string]any) string {
	xui, ok := prop["x-ui"].(map[string]any)
	if !ok {
		return ""
	}
	w, _ := xui["widget"].(string)
	return w
}

// widgetOrDefault returns the widget name or "text" (the default) if empty.
func widgetOrDefault(w string) string {
	if w == "" {
		return "text (default)"
	}
	return w
}

// containsAny returns true if s contains any of the substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
