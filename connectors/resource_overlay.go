package connectors

import "strings"

// ResourceRef is one opaque parameter ID resolved to a display name and optional URL.
type ResourceRef struct {
	Param string
	ID    string
	Name  string
	URL   string
}

// ResourceNameKey returns the overlay key for a parameter (`spreadsheet_id` → `spreadsheet_name`).
func ResourceNameKey(param string) string {
	if strings.HasSuffix(param, "_id") {
		return strings.TrimSuffix(param, "_id") + "_name"
	}
	return param + "_name"
}

// ResourceURLKey returns the overlay URL key for a parameter (`spreadsheet_id` → `spreadsheet_url`).
func ResourceURLKey(param string) string {
	if strings.HasSuffix(param, "_id") {
		return strings.TrimSuffix(param, "_id") + "_url"
	}
	return param + "_url"
}

// AttachResources adds overlay-aligned `{param}_name` / `{param}_url` fields and an
// ID-keyed `resources` map so UIs can resolve one or many IDs per parameter.
// Existing keys in details are left unchanged. Empty name/id refs are skipped.
func AttachResources(details map[string]any, refs ...ResourceRef) map[string]any {
	if details == nil {
		details = map[string]any{}
	}
	resources, _ := details["resources"].(map[string]any)
	if resources == nil {
		resources = map[string]any{}
	}
	added := false
	for _, ref := range refs {
		if ref.Param == "" || ref.ID == "" || ref.Name == "" {
			continue
		}
		nameKey := ResourceNameKey(ref.Param)
		if _, exists := details[nameKey]; !exists {
			details[nameKey] = ref.Name
		}
		if ref.URL != "" {
			urlKey := ResourceURLKey(ref.Param)
			if _, exists := details[urlKey]; !exists {
				details[urlKey] = ref.URL
			}
		}
		byID, _ := resources[ref.Param].(map[string]any)
		if byID == nil {
			byID = map[string]any{}
		}
		entry := map[string]any{"name": ref.Name}
		if ref.URL != "" {
			entry["url"] = ref.URL
		}
		byID[ref.ID] = entry
		resources[ref.Param] = byID
		added = true
	}
	if added {
		details["resources"] = resources
	}
	return details
}

// resourceNameAliases maps parameter keys to legacy flat resource_details keys
// so UIs can resolve names from details stored before overlay keys existed.
var resourceNameAliases = map[string][]string{
	"spreadsheet_id":  {"title"},
	"document_id":     {"title"},
	"presentation_id": {"presentation_title", "title"},
	"file_id":         {"file_name"},
	"item_id":         {"file_name", "document_title", "workbook_title", "presentation_title"},
	"folder_id":       {"folder_name"},
	"parent_id":       {"folder_name", "parent_name"},
	"drive_id":        {"folder_name"},
	"space_name":      {"space_display_name"},
	"channel":         {"channel_name"},
	"channel_id":      {"channel_name"},
	"event_id":        {"title"},
	"message_id":      {"subject"},
	"thread_id":       {"subject"},
	"calendar_id":     {"calendar_name"},
	"user_id":         {"user_name"},
}

// LookupResource returns the display name and optional URL for a parameter ID.
// Lookup order: resources[param][id], {param}_name/{param}_url, then legacy aliases.
func LookupResource(details map[string]any, param, id string) (name, url string, ok bool) {
	if len(details) == 0 || param == "" || id == "" {
		return "", "", false
	}
	if resources, _ := details["resources"].(map[string]any); resources != nil {
		if byID, _ := resources[param].(map[string]any); byID != nil {
			if entry, _ := byID[id].(map[string]any); entry != nil {
				name, _ = entry["name"].(string)
				url, _ = entry["url"].(string)
				if name != "" {
					return name, url, true
				}
			}
		}
	}
	nameKey := ResourceNameKey(param)
	if n, isStr := details[nameKey].(string); isStr && n != "" {
		urlKey := ResourceURLKey(param)
		u, _ := details[urlKey].(string)
		return n, u, true
	}
	for _, alias := range resourceNameAliases[param] {
		if n, isStr := details[alias].(string); isStr && n != "" {
			return n, "", true
		}
	}
	return "", "", false
}

// MergeResourceDetails copies keys from src into dst. Nested `resources` maps are merged.
func MergeResourceDetails(dst, src map[string]any) map[string]any {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = map[string]any{}
	}
	srcResources, srcHasResources := src["resources"].(map[string]any)
	for k, v := range src {
		if k == "resources" {
			continue
		}
		if _, exists := dst[k]; !exists {
			dst[k] = v
		}
	}
	if !srcHasResources {
		return dst
	}
	dstResources, _ := dst["resources"].(map[string]any)
	if dstResources == nil {
		dstResources = map[string]any{}
	}
	for param, raw := range srcResources {
		srcByID, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		dstByID, _ := dstResources[param].(map[string]any)
		if dstByID == nil {
			dstByID = map[string]any{}
		}
		for id, entry := range srcByID {
			if _, exists := dstByID[id]; !exists {
				dstByID[id] = entry
			}
		}
		dstResources[param] = dstByID
	}
	if len(dstResources) > 0 {
		dst["resources"] = dstResources
	}
	return dst
}
