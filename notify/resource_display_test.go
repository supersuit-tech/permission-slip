package notify

import "testing"

func TestOverlayParamDisplayName(t *testing.T) {
	t.Parallel()
	details := []byte(`{"resources":{"spreadsheet_id":{"s123":{"name":"Budget 2026"}}}}`)
	if got := overlayParamDisplayName(details, "spreadsheet_id", "s123"); got != "Budget 2026" {
		t.Errorf("got %q", got)
	}
	if got := overlayParamDisplayName(details, "spreadsheet_id", "missing"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestOverlayIDsInText(t *testing.T) {
	t.Parallel()
	details := []byte(`{"resources":{"spreadsheet_id":{"s123":{"name":"Budget 2026"}}}}`)
	got := overlayIDsInText(`"spreadsheet_id": "s123"`, details)
	if got != `"spreadsheet_id": "Budget 2026"` {
		t.Errorf("got %q", got)
	}
}
