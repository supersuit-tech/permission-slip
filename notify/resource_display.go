package notify

import (
	"encoding/json"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func overlayParamDisplayName(resourceDetails json.RawMessage, param, rawID string) string {
	details := unmarshalDetailsMap(resourceDetails)
	if details == nil {
		return ""
	}
	name, _, ok := connectors.LookupResource(details, param, rawID)
	if !ok || name == "" || name == rawID {
		return ""
	}
	return name
}

func unmarshalDetailsMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var details map[string]any
	if json.Unmarshal(raw, &details) != nil {
		return nil
	}
	return details
}

func overlayIDsInText(text string, resourceDetails json.RawMessage) string {
	details := unmarshalDetailsMap(resourceDetails)
	if details == nil || text == "" {
		return text
	}
	resources, _ := details["resources"].(map[string]any)
	if resources == nil {
		return text
	}
	type pair struct {
		id   string
		name string
	}
	var replacements []pair
	for _, raw := range resources {
		byID, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for id, entry := range byID {
			m, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			if id == "" || name == "" || name == id {
				continue
			}
			replacements = append(replacements, pair{id: id, name: name})
		}
	}
	// Replace longer IDs first so prefixes don't clobber.
	for i := 0; i < len(replacements); i++ {
		for j := i + 1; j < len(replacements); j++ {
			if len(replacements[j].id) > len(replacements[i].id) {
				replacements[i], replacements[j] = replacements[j], replacements[i]
			}
		}
	}
	out := text
	for _, r := range replacements {
		out = strings.ReplaceAll(out, r.id, r.name)
	}
	return out
}
