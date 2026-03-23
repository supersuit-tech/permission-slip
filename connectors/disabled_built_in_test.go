package connectors

import (
	"slices"
	"testing"
)

func TestDisabledBuiltInConnectorIDs_IncludesKrogerExcludesQuickBooks(t *testing.T) {
	t.Parallel()
	ids := DisabledBuiltInConnectorIDs()
	if !slices.Contains(ids, "kroger") {
		t.Errorf("expected disabled list to include %q, got %v", "kroger", ids)
	}
	if slices.Contains(ids, "quickbooks") {
		t.Errorf("expected disabled list to exclude %q (connector re-enabled), got %v", "quickbooks", ids)
	}
}
