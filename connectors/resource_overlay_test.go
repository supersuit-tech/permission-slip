package connectors

import "testing"

func TestAttachResources_OverlayKeysAndMap(t *testing.T) {
	details := map[string]any{"title": "Budget 2026"}
	got := AttachResources(details, ResourceRef{
		Param: "spreadsheet_id",
		ID:    "s123",
		Name:  "Budget 2026",
		URL:   "https://docs.google.com/spreadsheets/d/s123",
	})
	if got["spreadsheet_name"] != "Budget 2026" {
		t.Errorf("spreadsheet_name: got %v", got["spreadsheet_name"])
	}
	if got["spreadsheet_url"] != "https://docs.google.com/spreadsheets/d/s123" {
		t.Errorf("spreadsheet_url: got %v", got["spreadsheet_url"])
	}
	if got["title"] != "Budget 2026" {
		t.Errorf("title should be preserved, got %v", got["title"])
	}
	resources, _ := got["resources"].(map[string]any)
	byID, _ := resources["spreadsheet_id"].(map[string]any)
	entry, _ := byID["s123"].(map[string]any)
	if entry["name"] != "Budget 2026" {
		t.Errorf("resources map name: got %v", entry)
	}
}

func TestAttachResources_SkipsEmpty(t *testing.T) {
	got := AttachResources(nil, ResourceRef{Param: "folder_id", ID: "x", Name: ""})
	if _, ok := got["resources"]; ok {
		t.Errorf("expected no resources for empty name, got %#v", got)
	}
}

func TestAttachResources_DoesNotOverwriteExistingName(t *testing.T) {
	details := map[string]any{"folder_name": "Existing"}
	got := AttachResources(details, ResourceRef{
		Param: "folder_id",
		ID:    "f1",
		Name:  "New",
		URL:   "https://drive.google.com/drive/folders/f1",
	})
	if got["folder_name"] != "Existing" {
		t.Errorf("should keep existing folder_name, got %v", got["folder_name"])
	}
	if got["folder_url"] != "https://drive.google.com/drive/folders/f1" {
		t.Errorf("folder_url: got %v", got["folder_url"])
	}
}

func TestMergeResourceDetails_MergesResourcesMap(t *testing.T) {
	a := AttachResources(nil, ResourceRef{Param: "folder_id", ID: "a", Name: "A"})
	b := AttachResources(nil, ResourceRef{Param: "folder_id", ID: "b", Name: "B"})
	got := MergeResourceDetails(a, b)
	resources, _ := got["resources"].(map[string]any)
	byID, _ := resources["folder_id"].(map[string]any)
	if _, ok := byID["a"]; !ok {
		t.Errorf("missing folder a: %#v", byID)
	}
	if _, ok := byID["b"]; !ok {
		t.Errorf("missing folder b: %#v", byID)
	}
}

func TestResourceNameKey(t *testing.T) {
	if ResourceNameKey("spreadsheet_id") != "spreadsheet_name" {
		t.Errorf("spreadsheet_id: got %q", ResourceNameKey("spreadsheet_id"))
	}
	if ResourceNameKey("channel") != "channel_name" {
		t.Errorf("channel: got %q", ResourceNameKey("channel"))
	}
}

func TestLookupResource_ResourcesMapThenAlias(t *testing.T) {
	details := AttachResources(map[string]any{"title": "Budget 2026"}, ResourceRef{
		Param: "spreadsheet_id",
		ID:    "s123",
		Name:  "Budget 2026",
		URL:   "https://docs.google.com/spreadsheets/d/s123",
	})
	name, url, ok := LookupResource(details, "spreadsheet_id", "s123")
	if !ok || name != "Budget 2026" || url != "https://docs.google.com/spreadsheets/d/s123" {
		t.Errorf("resources map lookup: name=%q url=%q ok=%v", name, url, ok)
	}

	legacy := map[string]any{"folder_name": "Finance Shared Drive"}
	name, url, ok = LookupResource(legacy, "folder_id", "abc")
	if !ok || name != "Finance Shared Drive" || url != "" {
		t.Errorf("alias lookup: name=%q url=%q ok=%v", name, url, ok)
	}

	if _, _, ok := LookupResource(nil, "folder_id", "abc"); ok {
		t.Error("expected miss on nil details")
	}
}
