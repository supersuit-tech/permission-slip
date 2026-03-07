package kroger

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestID(t *testing.T) {
	t.Parallel()

	conn := New()
	if got := conn.ID(); got != "kroger" {
		t.Errorf("ID() = %q, want %q", got, "kroger")
	}
}

func TestActions(t *testing.T) {
	t.Parallel()

	conn := New()
	actions := conn.Actions()

	expected := []string{
		"kroger.search_products",
		"kroger.get_product",
		"kroger.search_locations",
		"kroger.add_to_cart",
	}
	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("Actions() missing %q", name)
		}
	}
	if len(actions) != len(expected) {
		t.Errorf("Actions() has %d entries, want %d", len(actions), len(expected))
	}
}

func TestValidateCredentials_Valid(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.ValidateCredentials(t.Context(), validCreds())
	if err != nil {
		t.Errorf("ValidateCredentials() unexpected error: %v", err)
	}
}

func TestValidateCredentials_Missing(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{}))
	if err == nil {
		t.Fatal("ValidateCredentials() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestValidateCredentials_Empty(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{
		"access_token": "",
	}))
	if err == nil {
		t.Fatal("ValidateCredentials() expected error, got nil")
	}
}

func TestManifest_Valid(t *testing.T) {
	t.Parallel()

	conn := New()
	manifest := conn.Manifest()

	if err := manifest.Validate(); err != nil {
		t.Errorf("Manifest().Validate() error: %v", err)
	}
}
