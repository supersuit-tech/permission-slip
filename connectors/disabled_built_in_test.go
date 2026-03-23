package connectors

import (
	"slices"
	"testing"
)

func TestDisabledBuiltInConnectorIDs_IncludesKrogerQuickBooksSalesforce(t *testing.T) {
	t.Parallel()
	ids := DisabledBuiltInConnectorIDs()
	for _, want := range []string{"kroger", "quickbooks", "salesforce"} {
		if !slices.Contains(ids, want) {
			t.Errorf("expected disabled list to include %q, got %v", want, ids)
		}
	}
}
